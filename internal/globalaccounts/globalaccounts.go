package globalaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
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

	logic(cfg, svc, kcp, db, logs)
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
	)

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

func logic(config Config, svc *http.Client, kcp client.Client, db storage.BrokerStorage, logs *logrus.Logger) {
	var okCount, getInstanceErrorCounts, requestErrorCount, mismatch, kebInstanceMissingSACount, kebInstanceMissingGACount, svcGlobalAccountMissing int
	var instanceUpdateErrorCount, labelsUpdateErrorCount int
	var mismatches []string
	labeler := broker.NewLabeler(kcp)

	instances, instancesCount, _, err := db.Instances().List(dbmodel.InstanceFilter{})
	if err != nil {
		logs.Errorf("while getting instances %s", err.Error())
		return
	}
	for i, instance := range instances {
		logs.Infof("instance i: %s r: %s %d/%d", instance.InstanceID, instance.RuntimeID, i+1, instancesCount)
		if instance.SubAccountID == "" {
			logs.Errorf("instance r: %s have empty SA %s", instance.RuntimeID, instance.SubAccountID)
			kebInstanceMissingSACount++
			continue
		}
		if instance.GlobalAccountID == "" {
			logs.Errorf("instance r: %s have empty GA %s", instance.RuntimeID, instance.GlobalAccountID)
			kebInstanceMissingGACount++
			continue
		}
		svcResponse, err := svcRequest(config, svc, instance.SubAccountID, logs)
		if err != nil {
			logs.Error(err.Error())
			requestErrorCount++
			continue
		}
		svcGlobalAccountId := svcResponse.GlobalAccountGUID

		if svcGlobalAccountId == "" {
			logs.Errorf("svc response is empty for %s", instance.InstanceID)
			svcGlobalAccountMissing++
			continue
		} else if svcGlobalAccountId != instance.GlobalAccountID {
			info := fmt.Sprintf("(INSTANCE i: %s r: %s MISMATCH) for subaccount %s is %s but it should be: %s", instance.InstanceID, instance.RuntimeID, instance.SubAccountID, instance.GlobalAccountID, svcGlobalAccountId)
			mismatches = append(mismatches, info)
			mismatch++
		} else {
			okCount++
			continue
		}

		if config.DryRun {
			logs.Infof("dry run: update instance in db %s with new %s", instance.InstanceID, svcGlobalAccountId)
			continue
		}

		instanceUpdateFail, labelsUpdateFail := updateData(&instance, svcGlobalAccountId, logs, *labeler, db)
		if instanceUpdateFail {
			instanceUpdateErrorCount++
		}
		if labelsUpdateFail {
			labelsUpdateErrorCount++
		}
	}

	showReport(logs, okCount, mismatch, getInstanceErrorCounts, kebInstanceMissingSACount, kebInstanceMissingGACount, requestErrorCount, instanceUpdateErrorCount, labelsUpdateErrorCount, instancesCount, svcGlobalAccountMissing, mismatches)
}

func updateData(instance *internal.Instance, svcGlobalAccountId string, logs *logrus.Logger, labeler broker.Labeler, db storage.BrokerStorage) (instanceUpdateFail bool, labelsUpdateFail bool) {
	if instance.SubscriptionGlobalAccountID == "" {
		instance.SubscriptionGlobalAccountID = instance.GlobalAccountID
	}
	instance.GlobalAccountID = svcGlobalAccountId
	_, err := db.Instances().Update(*instance)
	if err != nil {
		logs.Errorf("while updating db %s", err)
		instanceUpdateFail = true
		return
	}

	// isExpired checks if field expireAt is not empty, if yes, then it means it is suspended
	if instance.IsExpired() {
		logs.Infof("instance r: %s is suspended, skipping labels update", instance.RuntimeID)
		return
	}

	err = labeler.UpdateLabels(instance.RuntimeID, svcGlobalAccountId)
	if err != nil {
		logs.Errorf("while updating labels %s", err)
		labelsUpdateFail = true
		return
	}

	return
}

func showReport(logs *logrus.Logger, okCount, mismatch, getInstanceErrorCounts, kebInstanceMissingSACount, kebInstanceMissingGACount, requestErrorCount, instanceUpdateErrorCount, labelsUpdateErrorCount, instancesIDs, svcGlobalAccountMissing int, mismatches []string) {
	logs.Info("######## STATS ########")
	logs.Info("-----------------------")
	logs.Infof("total no. KEB instances: %d", instancesIDs)
	logs.Infof("=> OK: %d", okCount)
	logs.Infof("=> GA from KEB and GA from SVC are different: %d", mismatch)
	logs.Info("-----------------------")
	logs.Infof("no. instances in KEB which failed to get from db: %d", getInstanceErrorCounts)
	logs.Infof("no. instances in KEB with empty SA: %d", kebInstanceMissingSACount)
	logs.Infof("no. instances in KEB with empty GA: %d", kebInstanceMissingGACount)
	logs.Infof("no. GA missing in account service: %d", svcGlobalAccountMissing)
	logs.Infof("no. failed requests to account service : %d", requestErrorCount)
	logs.Infof("no. instances with error while updating in : %d", instanceUpdateErrorCount)
	logs.Infof("no. CR for which update labels failed: %d", labelsUpdateErrorCount)
	logs.Info("######## MISMATCHES ########")
	for _, mismatch := range mismatches {
		logs.Info(mismatch)
	}
	logs.Info("############################")
}
