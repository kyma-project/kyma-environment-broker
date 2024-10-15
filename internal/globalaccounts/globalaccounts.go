package globalaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
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

	cfg.DryRun = true // temporary set until program while analyze
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

	logic(cfg, svc, kcp, connection, db, logs)
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

func logic(config Config, svc *http.Client, kcp client.Client, connection *dbr.Connection, db storage.BrokerStorage, logs *logrus.Logger) {
	var okCount, getInstanceErrorCounts, requestErrorCount, mismatch, kebInstanceMissingSACount, kebInstanceMissingGACount, dbEmptyGA int
	var instanceUpdateErrorCount, labelsUpdateErrorCount int
	var out strings.Builder
	labeler := broker.NewLabeler(kcp)

	instancesIDs := make([]string, 0)
	_ = connection.QueryRow("select instance_id from instances").Scan(&instancesIDs) // suspended ones
	for _, instanceID := range instancesIDs {
		instance, err := db.Instances().GetByID(instanceID)
		if err != nil {
			logs.Errorf("while getting instance %s %s", instance.RuntimeID, err.Error())
			getInstanceErrorCounts++
			continue
		}
		if instance.SubAccountID == "" {
			logs.Errorf("instance have empty SA %s", instance.SubAccountID)
			kebInstanceMissingSACount++
			continue
		}
		if instance.GlobalAccountID == "" {
			logs.Errorf("instance have empty GA %s", instance.GlobalAccountID)
			kebInstanceMissingGACount++
			continue
		}
		svcResponse, err := svcRequest(config, svc, instanceID, logs)
		if err != nil {
			logs.Error(err.Error())
			requestErrorCount++
			continue
		}
		needFix := false
		if svcResponse.GlobalAccountGUID != instance.GlobalAccountID {
			needFix = true
			info := fmt.Sprintf("(INSTANCE MISSMATCH) for subaccount %s is %s but it should be: %s", instance.SubAccountID, instance.GlobalAccountID, svcResponse.GlobalAccountGUID)
			out.WriteString(info)
			mismatch++
		} else {
			okCount++
		}

		if needFix {
			if config.DryRun {
				logs.Infof("dry run: update instance in db %s with new %s", instance.RuntimeID, svcResponse.GlobalAccountGUID)
			} else {
				if instance.SubscriptionGlobalAccountID != "" {
					instance.SubscriptionGlobalAccountID = instance.GlobalAccountID
				}
				instance.GlobalAccountID = svcResponse.GlobalAccountGUID
				_, err := db.Instances().Update(*instance)
				if err != nil {
					logs.Errorf("while updating db %s", err)
					instanceUpdateErrorCount++
					continue
				}

				if !instance.IsExpired() { // expired instances have no CRs on KCP, so there is nothing to update
					err = labeler.UpdateLabels(instance.RuntimeID, svcResponse.GlobalAccountGUID)
					if err != nil {
						logs.Errorf("while updating labels %s", err)
						labelsUpdateErrorCount++
						continue
					}
				}
			}
		}
	}

	logs.Info("######## STATS ########")
	logs.Info("------------------------")
	logs.Infof("total no. KEB instances: %d", len(instancesIDs))
	logs.Infof("=> OK: %d", okCount)
	logs.Infof("=> GA from KEB and GA from SVC are different: %d", mismatch)
	logs.Info("------------------------")
	logs.Info("no instances in KEB which failed to get from db: %d", getInstanceErrorCounts)
	logs.Infof("no. instances in KEB with empty SA: %d", kebInstanceMissingSACount)
	logs.Infof("no. instances in KEB with empty GA: %d", kebInstanceMissingGACount)
	logs.Infof("no. failed requests to account service : %d", requestErrorCount)
	logs.Infof("no. instances with error while updating in : %d", instanceUpdateErrorCount)
	logs.Infof("no. CR for which update labels failed: %d", labelsUpdateErrorCount)
	logs.Info("######## MISMATCHES ########")
	logs.Info(out.String())
	logs.Info("############################")
}
