package globalaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/k8s"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	k8scfg "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const subaccountServicePath = "%s/accounts/v1/technical/subaccounts/%s"

type result struct {
	GlobalAccountGUID string `json:"globalAccountGUID"`
}

type svcConfig struct {
	ClientID       string
	ClientSecret   string
	AuthURL        string
	SubaccountsURL string
}

type fixMap struct {
	instance               internal.Instance
	correctGlobalAccountId string
}

func Run(ctx context.Context, cfg Config) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered: %s\n", r)
		}
	}()

	logs := logrus.New()
	logs.Infof("*** Start at: %s ***", time.Now().Format(time.RFC3339))
	logs.Infof("is dry run?: %t ", cfg.DryRun)

	svc, db, connection, kcp, err := initAll(ctx, cfg, logs)
	fatalOnError(err, logs)
	defer func() {
		err = connection.Close()
		if err != nil {
			logs.Error(err)
		}
	}()

	clusterOp, err := clusterOp(ctx, kcp, logs)
	fatalOnError(err, logs)
	logs.Println(fmt.Sprintf("No. kymas: %d", len(clusterOp.Items)))

	toFix := logic(cfg, svc, db, clusterOp, logs)
	fixGlobalAccounts(db.Instances(), kcp, cfg, toFix, logs)
	logs.Infof("*** End at: %s ***", time.Now().Format(time.RFC3339))
	<-ctx.Done()
}

func initAll(ctx context.Context, cfg Config, logs *logrus.Logger) (*http.Client, storage.BrokerStorage, *dbr.Connection, client.Client, error) {

	oauthConfig := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.AuthURL,
	}

	db, connection, err := storage.NewFromConfig(
		cfg.Database,
		events.Config{},
		storage.NewEncrypter(cfg.Database.SecretKey),
		logs.WithField("service", "storage"))

	if err != nil {
		logs.Error(err.Error())
		return nil, nil, nil, nil, err
	}

	kcpK8sClient, err := getKcpClient()
	if err != nil {
		logs.Error(err.Error())
		return nil, nil, nil, nil, err
	}

	svc := oauthConfig.Client(ctx)
	return svc, db, connection, kcpK8sClient, nil
}

func fatalOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}

func getKcpClient() (client.Client, error) {
	kcpK8sConfig, err := k8scfg.GetConfig()
	mapper, err := apiutil.NewDiscoveryRESTMapper(kcpK8sConfig)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDiscoveryRESTMapper(kcpK8sConfig)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("while waiting for client mapper: %w", err)
		}
	}
	cli, err := client.New(kcpK8sConfig, client.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("while creating a client: %w", err)
	}
	return cli, nil
}

func clusterOp(ctx context.Context, kcp client.Client, logs *logrus.Logger) (unstructured.UnstructuredList, error) {
	gvk, err := k8s.GvkByName(k8s.KymaCr)
	if err != nil {
		logs.Errorf("error getting GVK %s", err)
		return unstructured.UnstructuredList{}, nil
	}

	kymas := unstructured.UnstructuredList{}
	kymas.SetGroupVersionKind(gvk)
	err = kcp.List(ctx, &kymas, client.InNamespace("kcp-system"))
	if err != nil {
		logs.Errorf("error listing kyma %s", err)
		return unstructured.UnstructuredList{}, err
	}
	return kymas, nil
}

func dbOp(runtimeId string, db storage.BrokerStorage, logs *logrus.Logger) (internal.Instance, error) {
	runtimeIDFilter := dbmodel.InstanceFilter{RuntimeIDs: []string{runtimeId}}

	instances, _, _, err := db.Instances().List(runtimeIDFilter)
	if err != nil {
		logs.Error(err)
		return internal.Instance{}, err
	}
	if len(instances) == 0 {
		logs.Errorf("no instance for runtime id %s", runtimeId)
		return internal.Instance{}, fmt.Errorf("no instance for runtime id")
	}
	if len(instances) > 1 {
		logs.Errorf("more than one instance for runtime id %s", runtimeId)
		return internal.Instance{}, fmt.Errorf("more than one instance for runtime")
	}
	return instances[0], nil
}

func logic(config Config, svc *http.Client, db storage.BrokerStorage, kymas unstructured.UnstructuredList, logs *logrus.Logger) []fixMap {
	var resOk, dbErrors, reqErrors, resEmptyGA, resWrongGa, dbEmptySA, dbEmptyGA int
	var out strings.Builder
	toFix := make([]fixMap, 0)
	for i, kyma := range kymas.Items {
		runtimeId := kyma.GetName() // name of kyma is runtime id
		fmt.Printf("proccessings %d/%d : %s \n", i, len(kymas.Items), runtimeId)
		dbOp, err := dbOp(runtimeId, db, logs)
		if err != nil {
			logs.Errorf("error getting data from db %s", err)
			dbErrors++
			continue
		}

		if dbOp.SubAccountID == "" {
			logs.Errorf("instance have empty SA %s", dbOp.SubAccountID)
			dbEmptySA++
			continue
		}
		if dbOp.GlobalAccountID == "" {
			logs.Errorf("instance have empty GA %s", dbOp.GlobalAccountID)
			dbEmptyGA++
			continue
		}

		svcResponse, err := svcRequest(config, svc, dbOp.SubAccountID, logs)
		if err != nil {
			logs.Errorf("error requesting %s", err)
			reqErrors++
			continue
		}

		info := ""
		switch {
		case svcResponse.GlobalAccountGUID == "":
			info = fmt.Sprintf(" [EMPTY] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", dbOp.SubAccountID, dbOp.GlobalAccountID, svcResponse.GlobalAccountGUID)
			resEmptyGA++
		case svcResponse.GlobalAccountGUID != dbOp.GlobalAccountID:
			info = fmt.Sprintf(" [WRONG] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", dbOp.SubAccountID, dbOp.GlobalAccountID, svcResponse.GlobalAccountGUID)
			toFix = append(toFix, fixMap{instance: dbOp, correctGlobalAccountId: dbOp.GlobalAccountID})
			resWrongGa++
		default:
			resOk++
		}

		if info != "" {
			out.WriteString(info)
		}
	}

	logs.Info("\n\n")
	logs.Info("######## stats ########")
	logs.Infof("ok: %d \n", resOk)
	logs.Infof("dbErrors: %d \n", dbErrors)
	logs.Infof("db emty SA: %d \n", dbEmptySA)
	logs.Infof("db emty GA: %d \n", dbEmptyGA)
	logs.Infof("reqErrors: %d \n", reqErrors)
	logs.Infof("emptyGA: %d \n", resEmptyGA)
	logs.Infof("wrongGa: %d \n", resWrongGa)
	logs.Info("########################")
	logs.Info("######## to fix ########")
	logs.Info(out.String())
	logs.Info("########################")
	logs.Info("\n\n")

	return toFix
}

func svcRequest(config Config, svc *http.Client, subaccountId string, logs *logrus.Logger) (result, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(subaccountServicePath, config.ServiceURL, subaccountId), nil)
	if err != nil {
		logs.Errorf("while creating request %s", err)
		return result{}, err
	}
	query := request.URL.Query()
	request.URL.RawQuery = query.Encode()
	response, err := svc.Do(request)
	if err != nil {
		logs.Errorf("svc response error: %s", err.Error())
		return result{}, err
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			logs.Errorf("while closing body: %s", err.Error())
		}
	}()
	if response.StatusCode != http.StatusOK {
		return result{}, fmt.Errorf("url: %s : response status -> %s", request.URL, response.Status)
	}
	var svcResponse result
	err = json.NewDecoder(response.Body).Decode(&svcResponse)
	if err != nil {
		logs.Errorf("while decoding response: %s", err.Error())
		return result{}, err
	}
	return svcResponse, nil
}

func fixGlobalAccounts(db storage.Instances, kcp client.Client, cfg Config, toFix []fixMap, logs *logrus.Logger) {
	_ = broker.NewLabeler(kcp)
	updateErrorCounts := 0
	processed := 0
	logs.Infof("fix start. Is dry run?: %t", cfg.DryRun)
	for _, pair := range toFix {
		processed++
		if cfg.DryRun {
			logs.Infof("dry run: update labels for runtime %s with new %s", pair.instance.RuntimeID, pair.correctGlobalAccountId)
			continue
		}
		if cfg.Probe > -1 && (processed >= cfg.Probe) {
			logs.Infof("processed probe of %d instances", processed)
			break
		}

		/*if pair.instance.SubscriptionGlobalAccountID != "" {
			pair.instance.SubscriptionGlobalAccountID = pair.instance.GlobalAccountID
		}
		pair.instance.GlobalAccountID = pair.correctGlobalAccountId
		newInstance, err := db.Update(pair.instance)
		if err != nil {
			logs.Errorf("error updating db %s", err)
			errs++
			continue
		}
		err = labeler.UpdateLabels(newInstance.RuntimeID, pair.correctGlobalAccountId)
		if err != nil {
			logs.Errorf("error updating labels %s", err)
			errs++
			continue
		}*/
	}

	if updateErrorCounts > 0 {
		logs.Infof("finished update with %d errors", updateErrorCounts)
	} else {
		logs.Info("finished update with no error")
	}
}
