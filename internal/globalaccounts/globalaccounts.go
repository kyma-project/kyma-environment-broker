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

type svcResult struct {
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
	label                  bool
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

	toFix := logic(cfg, svc, connection, db, clusterOp, logs)
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

func logic(config Config, svc *http.Client, connection *dbr.Connection, db storage.BrokerStorage, kymas unstructured.UnstructuredList, logs *logrus.Logger) []fixMap {
	var resOk, dbErrors, reqErrors, instanceMissmatch, suspendendMissmatch, dbEmptySA, dbEmptyGA int
	var out strings.Builder
	toFix := make([]fixMap, 0)
	for i, kyma := range kymas.Items {
		runtimeId := kyma.GetName() // name of kyma is runtime id
		logs.Printf("proccessings %d/%d : %s \n", i, len(kymas.Items), runtimeId)
		instance, err := dbOp(runtimeId, db, logs)
		if err != nil {
			logs.Errorf("error getting data from db %s", err)
			dbErrors++
			continue
		}

		if instance.SubAccountID == "" {
			logs.Errorf("instance have empty SA %s", instance.SubAccountID)
			dbEmptySA++
			continue
		}
		if instance.GlobalAccountID == "" {
			logs.Errorf("instance have empty GA %s", instance.GlobalAccountID)
			dbEmptyGA++
			continue
		}

		svcResponse, err := svcRequest(config, svc, instance.SubAccountID, logs)
		if err != nil {
			logs.Errorf("error requesting %s", err)
			reqErrors++
			continue
		}

		if svcResponse.GlobalAccountGUID != instance.GlobalAccountID {
			info := fmt.Sprintf("(INSTANCE MISSMATCH) for subaccount %s is %s but it should be: %s", instance.SubAccountID, instance.GlobalAccountID, svcResponse.GlobalAccountGUID)
			out.WriteString(info)
			toFix = append(toFix, fixMap{instance: instance, correctGlobalAccountId: instance.GlobalAccountID, label: true})
			instanceMissmatch++
		} else {
			resOk++
		}
	}

	// if there is no runtime_id in instances table it means that instance is suspended
	noRuntimes := make([]string, 0)
	_ = connection.QueryRow("select instance_id from instances where runtime_id = ''").Scan(&noRuntimes) // suspended ones
	for _, instanceID := range noRuntimes {
		instance, err := db.Instances().GetByID(instanceID)
		if err != nil {
			logs.Errorf("while getting instance %s %s", instance.RuntimeID, err.Error())
			dbErrors++
			continue
		}
		svcResponse, err := svcRequest(config, svc, instanceID, logs)
		if err != nil {
			reqErrors++
			continue
		}
		if svcResponse.GlobalAccountGUID != instance.GlobalAccountID {
			info := fmt.Sprintf("(SUSPENDED MISSMATCH) for subaccount %s is %s but it should be: %s", instance.SubAccountID, instance.GlobalAccountID, svcResponse.GlobalAccountGUID)
			out.WriteString(info)
			toFix = append(toFix, fixMap{instance: *instance, correctGlobalAccountId: instance.GlobalAccountID, label: false})
			suspendendMissmatch++
		} else {
			resOk++
		}
	}

	logs.Info("######## stats ########")
	logs.Infof("total: %d", len(kymas.Items))
	logs.Infof("=> ok: %d", resOk)
	logs.Infof("=> instances not ok: %d", instanceMissmatch)
	logs.Infof("=> suspended not ok: %d", suspendendMissmatch)
	logs.Infof("=> db empty SA: %d", dbEmptySA)
	logs.Infof("==> db empty GA: %d", dbEmptyGA)
	logs.Infof("==> db error: %d", dbErrors)
	logs.Infof("==> request error: %d", reqErrors)
	logs.Info("########################")
	logs.Info("######## to-fix ########")
	logs.Info(out.String())
	logs.Info("########################")

	return toFix
}

func svcRequest(config Config, svc *http.Client, subaccountId string, logs *logrus.Logger) (svcResult, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(subaccountServicePath, config.ServiceURL, subaccountId), nil)
	if err != nil {
		logs.Errorf("while creating request %s", err)
		return svcResult{}, err
	}
	query := request.URL.Query()
	request.URL.RawQuery = query.Encode()
	response, err := svc.Do(request)
	if err != nil {
		logs.Errorf("while doing request: %s", err.Error())
		return svcResult{}, err
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			logs.Errorf("while closing body: %s", err.Error())
		}
	}()
	if response.StatusCode != http.StatusOK {
		return svcResult{}, fmt.Errorf("while fail on url: %s : due to response status -> %s", request.URL, response.Status)
	}
	var svcResponse svcResult
	err = json.NewDecoder(response.Body).Decode(&svcResponse)
	if err != nil {
		logs.Errorf("while decoding response: %s", err.Error())
		return svcResult{}, err
	}
	return svcResponse, nil
}

func fixGlobalAccounts(db storage.Instances, kcp client.Client, cfg Config, toFix []fixMap, logs *logrus.Logger) {
	labeler := broker.NewLabeler(kcp)
	updateErrorCounts := 0
	processed := 0
	logs.Infof("fixGlobalAccounts func start. Is dry run?: %t", cfg.DryRun)
	for _, fixMap := range toFix {
		processed++
		if processed == cfg.Probe {
			logs.Infof("processed probe of %d instances", processed)
			break
		}
		instance := fixMap.instance
		if cfg.DryRun {
			logs.Infof("dry run: update labels for runtime %s with new %s", instance.RuntimeID, fixMap.correctGlobalAccountId)
			continue
		}

		if instance.SubscriptionGlobalAccountID != "" {
			instance.SubscriptionGlobalAccountID = instance.GlobalAccountID
		}
		instance.GlobalAccountID = fixMap.correctGlobalAccountId
		_, err := db.Update(instance)
		if err != nil {
			logs.Errorf("while updating db %s", err)
			updateErrorCounts++
			continue
		}

		// we fix labels when instance is not suspended, because if it is then there is no CRs
		if fixMap.label {
			err = labeler.UpdateLabels(instance.RuntimeID, fixMap.correctGlobalAccountId)
			if err != nil {
				logs.Errorf("while updating labels %s", err)
				updateErrorCounts++
			}
		}
	}

	if updateErrorCounts > 0 {
		logs.Infof("fixGlobalAccounts finished update with %d errors", updateErrorCounts)
	} else {
		logs.Info("fixGlobalAccounts finished update with no error")
	}
}
