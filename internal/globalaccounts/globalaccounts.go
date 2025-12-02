package globalaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gocraft/dbr"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/broker"
	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info(fmt.Sprintf("*** Start at: %s ***", time.Now().Format(time.RFC3339)))
	slog.Info(fmt.Sprintf("is dry run?: %t", cfg.DryRun))

	svc, db, connection, kcp, err := initAll(ctx, cfg)

	fatalOnError(err)
	defer func() {
		err = connection.Close()
		if err != nil {
			slog.Error(err.Error())
		}
	}()

	logic(cfg, svc, kcp, db)
	slog.Info(fmt.Sprintf("*** End at: %s ***", time.Now().Format(time.RFC3339)))

	<-ctx.Done()
}

func initAll(ctx context.Context, cfg Config) (*http.Client, storage.BrokerStorage, *dbr.Connection, client.Client, error) {

	oauthConfig := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.AuthURL,
	}

	db, connection, err := storage.NewFromConfig(
		cfg.Database,
		events.Config{},
		storage.NewEncrypter(cfg.Database.SecretKey, cfg.Database.Fips.WriteGcm),
	)

	if err != nil {
		slog.Error(err.Error())
		return nil, nil, nil, nil, err
	}

	kcpK8sClient, err := getKcpClient()
	if err != nil {
		slog.Error(err.Error())
		return nil, nil, nil, nil, err
	}

	svc := oauthConfig.Client(ctx)
	return svc, db, connection, kcpK8sClient, nil
}

func fatalOnError(err error) {
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func getKcpClient() (client.Client, error) {
	kcpK8sConfig, err := k8scfg.GetConfig()
	httpClient, err := rest.HTTPClientFor(kcpK8sConfig)
	if err != nil {
		return nil, fmt.Errorf("while creating HTTP client for REST mapper: %w", err)
	}
	mapper, err := apiutil.NewDynamicRESTMapper(kcpK8sConfig, httpClient)
	if err != nil {
		err = wait.PollUntilContextTimeout(context.Background(), time.Second, time.Minute, false, func(ctx context.Context) (bool, error) {
			mapper, err = apiutil.NewDynamicRESTMapper(kcpK8sConfig, httpClient)
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

func svcRequest(config Config, svc *http.Client, subaccountId string) (svcResult, error) {
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(subaccountServicePath, config.ServiceURL, subaccountId), nil)
	if err != nil {
		slog.Error(fmt.Sprintf("while creating request %s", err))
		return svcResult{}, err
	}
	query := request.URL.Query()
	request.URL.RawQuery = query.Encode()
	response, err := svc.Do(request)
	if err != nil {
		slog.Error(fmt.Sprintf("while doing request: %s", err.Error()))
		return svcResult{}, err
	}
	defer func() {
		err = response.Body.Close()
		if err != nil {
			slog.Error(fmt.Sprintf("while closing body: %s", err.Error()))
		}
	}()
	if response.StatusCode != http.StatusOK {
		return svcResult{}, fmt.Errorf("while fail on url: %s : due to response status -> %s", request.URL, response.Status)
	}
	var svcResponse svcResult
	err = json.NewDecoder(response.Body).Decode(&svcResponse)
	if err != nil {
		slog.Error(fmt.Sprintf("while decoding response: %s", err.Error()))
		return svcResult{}, err
	}
	return svcResponse, nil
}

func logic(config Config, svc *http.Client, kcp client.Client, db storage.BrokerStorage) {
	var okCount, getInstanceErrorCounts, requestErrorCount, mismatch, kebInstanceMissingSACount, kebInstanceMissingGACount, svcGlobalAccountMissing int
	var instanceUpdateErrorCount, labelsUpdateErrorCount int
	var mismatches []string
	labeler := broker.NewLabeler(kcp)

	instances, instancesCount, _, err := db.Instances().List(dbmodel.InstanceFilter{})
	if err != nil {
		slog.Error(fmt.Sprintf("while getting instances %s", err.Error()))
		return
	}
	for i, instance := range instances {
		slog.Info(fmt.Sprintf("instanceID: %s runtimeID: %s %d/%d", instance.InstanceID, instance.RuntimeID, i+1, instancesCount))
		if instance.SubAccountID == "" {
			slog.Error(fmt.Sprintf("instance with runtimeID: %s have empty SA %s", instance.RuntimeID, instance.SubAccountID))
			kebInstanceMissingSACount++
			continue
		}
		if instance.GlobalAccountID == "" {
			slog.Error(fmt.Sprintf("instance with runtimeID: %s have empty GA %s", instance.RuntimeID, instance.GlobalAccountID))
			kebInstanceMissingGACount++
			continue
		}
		svcResponse, err := svcRequest(config, svc, instance.SubAccountID)
		if err != nil {
			slog.Error(err.Error())
			requestErrorCount++
			continue
		}
		svcGlobalAccountId := svcResponse.GlobalAccountGUID

		if svcGlobalAccountId == "" {
			slog.Error(fmt.Sprintf("svc response is empty for instance %s", instance.InstanceID))
			svcGlobalAccountMissing++
			continue
		} else if svcGlobalAccountId != instance.GlobalAccountID {
			info := fmt.Sprintf("(instanceID: %s runtimeID: %s MISMATCH) for subaccount %s is %s but it should be: %s", instance.InstanceID, instance.RuntimeID, instance.SubAccountID, instance.GlobalAccountID, svcGlobalAccountId)
			mismatches = append(mismatches, info)
			mismatch++
		} else {
			okCount++
			continue
		}

		if config.DryRun {
			slog.Info(fmt.Sprintf("dry run: update instance %s in db with new global account %s", instance.InstanceID, svcGlobalAccountId))
			continue
		}

		instanceUpdateFail, labelsUpdateFail := updateData(&instance, svcGlobalAccountId, *labeler, db)
		if instanceUpdateFail {
			instanceUpdateErrorCount++
		}
		if labelsUpdateFail {
			labelsUpdateErrorCount++
		}
	}

	showReport(okCount, mismatch, getInstanceErrorCounts, kebInstanceMissingSACount, kebInstanceMissingGACount, requestErrorCount, instanceUpdateErrorCount, labelsUpdateErrorCount, instancesCount, svcGlobalAccountMissing, mismatches)
}

func updateData(instance *internal.Instance, svcGlobalAccountId string, labeler broker.Labeler, db storage.BrokerStorage) (instanceUpdateFail bool, labelsUpdateFail bool) {
	if instance.SubscriptionGlobalAccountID == "" {
		instance.SubscriptionGlobalAccountID = instance.GlobalAccountID
	}
	instance.GlobalAccountID = svcGlobalAccountId
	_, err := db.Instances().Update(*instance)
	if err != nil {
		slog.Error(fmt.Sprintf("while updating db %s", err))
		instanceUpdateFail = true
		return
	}

	// isExpired checks if field expireAt is not empty, if yes, then it means it is suspended
	if instance.IsExpired() {
		slog.Info(fmt.Sprintf("instance with runtimeID: %s is suspended, skipping labels update", instance.RuntimeID))
		return
	}

	err = labeler.UpdateLabels(instance.RuntimeID, svcGlobalAccountId)
	if err != nil {
		slog.Error(fmt.Sprintf("while updating labels %s", err))
		labelsUpdateFail = true
		return
	}

	return
}

func showReport(okCount, mismatch, getInstanceErrorCounts, kebInstanceMissingSACount, kebInstanceMissingGACount, requestErrorCount, instanceUpdateErrorCount, labelsUpdateErrorCount, instancesIDs, svcGlobalAccountMissing int, mismatches []string) {
	slog.Info("######## STATS ########")
	slog.Info("-----------------------")
	slog.Info(fmt.Sprintf("total no. KEB instances: %d", instancesIDs))
	slog.Info(fmt.Sprintf("=> OK: %d", okCount))
	slog.Info(fmt.Sprintf("=> GA from KEB and GA from SVC are different: %d", mismatch))
	slog.Info("-----------------------")
	slog.Info(fmt.Sprintf("no. instances in KEB which failed to get from db: %d", getInstanceErrorCounts))
	slog.Info(fmt.Sprintf("no. instances in KEB with empty SA: %d", kebInstanceMissingSACount))
	slog.Info(fmt.Sprintf("no. instances in KEB with empty GA: %d", kebInstanceMissingGACount))
	slog.Info(fmt.Sprintf("no. GA missing in account service: %d", svcGlobalAccountMissing))
	slog.Info(fmt.Sprintf("no. failed requests to account service: %d", requestErrorCount))
	slog.Info(fmt.Sprintf("no. instances with error while updating in: %d", instanceUpdateErrorCount))
	slog.Info(fmt.Sprintf("no. CR for which update labels failed: %d", labelsUpdateErrorCount))
	slog.Info("######## MISMATCHES ########")
	for _, mismatch := range mismatches {
		slog.Info(mismatch)
	}
	slog.Info("############################")
}
