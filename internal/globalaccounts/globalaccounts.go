package globalaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal"
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

type result struct {
	GlobalAccountGUID string `json:"globalAccountGUID"`
}

type svcConfig struct {
	ClientID       string
	ClientSecret   string
	AuthURL        string
	SubaccountsURL string
}

func Run(ctx context.Context, cfg Config) {
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

	logic(cfg, svc, db, clusterOp, logs)
	logs.Infof("*** End at: %s ***", time.Now().Format(time.RFC3339))
}

func initAll(ctx context.Context, cfg Config, logs *logrus.Logger) (*http.Client, storage.BrokerStorage, *dbr.Connection, client.Client, error) {
	svcConfig := svcConfig{
		ClientID:     cfg.AccountServiceID,
		ClientSecret: cfg.AccountServiceSecret,
		AuthURL:      cfg.AccountServiceURL,
	}

	oauthConfig := clientcredentials.Config{
		ClientID:     svcConfig.ClientID,
		ClientSecret: svcConfig.ClientSecret,
		TokenURL:     svcConfig.AuthURL,
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
	err = kcp.List(ctx, &kymas)
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

func logic(config Config, svc *http.Client, db storage.BrokerStorage, kymas unstructured.UnstructuredList, logs *logrus.Logger) {
	var resOk, dbErrors, reqErrors, resEmptyGA, resWrongGa, dbEmptySA, dbEmptyGA int
	for _, kyma := range kymas.Items {
		runtimeId := kyma.GetName() // name of kyma is runtime id
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

		switch {
		case svcResponse.GlobalAccountGUID == "":
			fmt.Printf(" [EMPTY] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", dbOp.SubAccountID, dbOp.GlobalAccountID, svcResponse.GlobalAccountGUID)
			resEmptyGA++
		case svcResponse.GlobalAccountGUID != dbOp.GlobalAccountID:
			fmt.Printf(" [WRONG] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", dbOp.SubAccountID, dbOp.GlobalAccountID, svcResponse.GlobalAccountGUID)
			resWrongGa++
		default:
			fmt.Printf(" [OK] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", dbOp.SubAccountID, dbOp.GlobalAccountID, svcResponse.GlobalAccountGUID)
			resOk++
		}
	}
	fmt.Printf("ok: %d \n", resOk)
	fmt.Printf("dbErrors: %d \n", dbErrors)
	fmt.Printf("db emty SA: %d \n", dbEmptySA)
	fmt.Printf("db emty GA: %d \n", dbEmptyGA)
	fmt.Printf("reqErrors: %d \n", reqErrors)
	fmt.Printf("emptyGA: %d \n", resEmptyGA)
	fmt.Printf("wrongGa: %d \n", resWrongGa)
}

func svcRequest(config Config, svc *http.Client, subaccountId string, logs *logrus.Logger) (result, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(config.AccountServiceURL, subaccountId), nil)
	if err != nil {
		logs.Errorf("error creating request %s", err)
		return result{}, err
	}
	query := request.URL.Query()
	request.URL.RawQuery = query.Encode()
	response, err := svc.Do(request)
	if err != nil {
		logs.Error(err)
		return result{}, err
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			logs.Error(err)
		}
	}()

	var svcResponse result
	err = json.NewDecoder(response.Body).Decode(&svcResponse)
	if err != nil {
		logs.Error(err.Error())
		return result{}, err
	}
	return svcResponse, nil
}
