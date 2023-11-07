# SAP BTP, Kyma runtime operations

Kyma Environment Broker (KEB) allows you to configure operations that you can run on a SAP BTP, Kyma runtime. Each operation consists of several steps and each step is represented by a separate file. As every step can be re-launched multiple times, for each step, you should determine a behavior in case of a processing failure. It can either:

- Return an error, which interrupts the entire process, or
- Repeat the entire operation after the specified period.

> **NOTE:** It's important to set lower timeouts for the Kyma installation in the Runtime Provisioner.

## Provision

The provisioning process is executed when the instance is created, or an unsuspension is triggered.
Each provisioning step is responsible for a separate part of preparing Kyma runtime. For example, in a step you can provide tokens, credentials, or URLs to integrate SAP BTP, Kyma runtime with external systems.
You can find all the provisioning steps in the [provisioning](../../cmd/broker/provisioning.go) file.

> **NOTE:** The timeout for processing this operation is set to `24h`.

## Deprovision

Each deprovisioning step is responsible for a separate part of cleaning Kyma runtime dependencies. To properly deprovision all the dependencies, you need the data used during the Kyma runtime provisioning. The first step finds the previous operation and copies the data.

None of the deprovisioning steps should block the entire deprovisioning operation. Use the `RetryOperationWithoutFail` function from the `DeprovisionOperationManager` struct to skip a step in case of a retry timeout. Set a 5-minute, at the most, timeout for retries in a step.
Only one step may fail the operation, namely `Check_Runtime_Removal`. It fails the operation in case of a timeout while checking for the Provisioner to remove the shoot. 
Once the step is successfully executed, it isn't retried (every deprovisioning step is defined in a separate stage). If a step has been skipped due to a retry timeout or error, the [Cron Job](../contributor/06-50-deprovision-retrigger-cronjob.md) tries to deprovision all remaining Kyma runtime dependencies again at a scheduled time.
You can find all the deprovisioning steps in the [deprovisioning](../../cmd/broker/deprovisioning.go) file.

> **NOTE:** The timeout for processing this operation is set to `24h`.

## Update

The update process is triggered by an [OSB API update operation](https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md#updating-a-service-instance) request.
You can find all the updating steps in the [update](../../cmd/broker/update.go) file.

## Upgrade cluster

The upgrade cluster process is triggered by upgrade cluster orchestration.
You can find all the upgrading cluster steps in the [upgrade_cluster](../../cmd/broker/upgrade_cluster.go) file.

## Upgrade Kyma

The upgrade Kyma process is triggered by upgrade Kyma orchestration.
You can find all the upgrading Kyma steps in the [upgrade_kyma](../../cmd/broker/upgrade_kyma.go) file.

## Provide additional steps

You can configure SAP BTP, Kyma runtime operations by providing additional steps. To add a new step, follow these tutorials:

<div tabs name="runtime-provisioning-deprovisioning" group="runtime-provisioning-deprovisioning">
  <details>
  <summary label="provisioning">
  Provisioning
  </summary>

1. Create a new file in [this directory](../../internal/process/provisioning).

2. Implement the following interface in your provisioning or deprovisioning step:

    ```go
    type Step interface {
        Name() string
        Run(operation internal.Operation, logger logrus.FieldLogger) (internal.Operation, time.Duration, error)
    }
    ```

   - `Name()` method returns the name of the step that is used in logs.
   - `Run()` method implements the functionality of the step. The method receives operations as an argument to which it can add appropriate overrides or save other used variables.


    ```go
    operation.InputCreator.SetOverrides(COMPONENT_NAME, []*gqlschema.ConfigEntryInput{
        {
            Key:   "path.to.key",
            Value: SOME_VALUE,
        },
        {
            Key:    "path.to.secret",
            Value:  SOME_VALUE,
            Secret: ptr.Bool(true),
        },
    })
    ```

    If your functionality contains long-term processes, you can store data in the storage.
    To do this, add the following field to the provisioning operation in which you want to save data:

    ```go
    type Operation struct {

        // These fields are serialized to JSON and stored in the storage
        RuntimeVersion RuntimeVersionData `json:"runtime_version"`

        // These fields are not stored in the storage
        InputCreator ProvisionerInputCreator `json:"-"`
    }
    ```

    By saving data in the storage, you can check if you already have the necessary data and avoid time-consuming processes. You must always return the modified operation from the method.

    See the example of the step implementation:

    ```go
    package provisioning

    import (
        "encoding/json"
        "net/http"
        "time"

        "github.com/kyma-incubator/compass/components/provisioner/pkg/gqlschema"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal/storage"

        "github.com/sirupsen/logrus"
    )

    type HelloWorldStep struct {
        operationStorage storage.Operations
        client           *http.Client
    }

    type ExternalBodyResponse struct {
        data  string
        token string
    }

    func NewHelloWorldStep(operationStorage storage.Operations, client *http.Client) *HelloWorldStep {
        return &HelloWorldStep{
            operationStorage: operationStorage,
            client:           client,
        }
    }

    func (s *HelloWorldStep) Name() string {
        return "Hello_World"
    }

    // Your step can be repeated in case any other step fails, even if your step has already done its job
    func (s *HelloWorldStep) Run(operation internal.Operation, log *logrus.Entry) (internal.Operation, time.Duration, error) {
        log.Info("Start step")

        // Check whether your step should be run or if its job has been done in the previous iteration
        // All non-save operation data are empty (e.g. InputCreator overrides)

        // Add your logic here

        // Add a call to an external service (optional)
        response, err := s.client.Get("http://example.com")
        if err != nil {
            // Error during a call to an external service may be temporary so you should return time.Duration
            // All steps will be repeated in X seconds/minutes
            return operation, 1 * time.Second, nil
        }
        defer response.Body.Close()

        body := ExternalBodyResponse{}
        err = json.NewDecoder(response.Body).Decode(&body)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If a call or any other action is time-consuming, you can save the result in the operation
        // If you need an extra field in the Operation structure, add it first
        // In the following step, you can check beforehand if a given value already exists in the operation
        operation.HelloWorlds = body.data
        updatedOperation, err := s.operationStorage.UpdateOperation(operation)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If your step finishes with data which should be added to override used during the Runtime provisioning,
        // add an extra value to operation.InputCreator, then return the updated version of the Application
        updatedOperation.InputCreator.SetOverrides("component-name", []*gqlschema.ConfigEntryInput{
            {
                Key:   "some.key",
                Value: body.token,
            },
        })

        // Return the updated version of the Application
        return *updatedOperation, 0, nil
    }
    ```

3. Add the step to the [`/cmd/broker/provisioning.go`](../../cmd/broker/provisioning.go) file:

    ```go
    provisioningSteps := []struct {
   		stage   string
   		step     provisioning.Step
   	}{
   		{
   			stage: "create_runtime",
   			step:   provisioning.NewHelloWorldStep(db.Operations(), &http.Client{}),
   		},
    }
    ```

   Once all the steps in the stage have run successfully, the stage is  not retried even if the application is restarted.

  </details>

  <details>
  <summary label="upgrade">
  Upgrade
  </summary>

1. Create a new file in [this directory](../../internal/process/upgrade_kyma).

2. Implement the following interface in your upgrade step:

    ```go
    type Step interface {
        Name() string
        Run(operation internal.UpgradeOperation, logger logrus.FieldLogger) (internal.UpgradeOperation, time.Duration, error)
    }
    ```

   - `Name()` method returns the name of the step that is used in logs.
   - `Run()` method implements the functionality of the step. The method receives operations as an argument to which it can add appropriate overrides or save other used variables.


    If your functionality contains long-term processes, you can store data in the storage.
    To do this, add this field to the upgrade operation in which you want to save data:

    ```go
    type UpgradeOperation struct {
        Operation `json:"-"`

        // add additional data here
    }
    ```

    By saving data in the storage, you can check if you already have the necessary data and avoid time-consuming processes. You should always return the modified operation from the method.

    See the example of the step implementation:

    ```go
    package upgrade

    import (
        "encoding/json"
        "net/http"
        "time"

        "github.com/kyma-incubator/compass/components/provisioner/pkg/gqlschema"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal"
        "github.com/kyma-incubator/compass/components/kyma-environment-broker/internal/storage"

        "github.com/sirupsen/logrus"
    )

    type HelloWorldStep struct {
        operationStorage storage.Operations
        client           *http.Client
    }

    type ExternalBodyResponse struct {
        data  string
        token string
    }

    func NewHelloWorldStep(operationStorage storage.Operations, client *http.Client) *HelloWorldStep {
        return &HelloWorldStep{
            operationStorage: operationStorage,
            client:           client,
        }
    }

    func (s *HelloWorldStep) Name() string {
        return "Hello_World"
    }

    // Your step can be repeated in case any other step fails, even if your step has already done its job
    func (s *HelloWorldStep) Run(operation internal.UpgradeOperation, log *logrus.Entry) (internal.UpgradeOperation, time.Duration, error) {
        log.Info("Start step")

        // Check whether your step should be run or if its job has been done in the previous iteration
        // All non-save operation data are empty (e.g. InputCreator overrides)

        // Add your logic here

        // Add a call to an external service (optional)
        response, err := s.client.Get("http://example.com")
        if err != nil {
            // Error during a call to an external service may be temporary so you should return time.Duration
            // All steps will be repeated in X seconds/minutes
            return operation, 1 * time.Second, nil
        }
        defer response.Body.Close()

        body := ExternalBodyResponse{}
        err = json.NewDecoder(response.Body).Decode(&body)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If a call or any other action is time-consuming, you can save the result in the operation
        // If you need an extra field in the UpgradeOperation structure, add it first
        // in the step below; beforehand, you can check if a given value already exists in the operation
        operation.HelloWorlds = body.data
        updatedOperation, err := s.operationStorage.UpdateUpgradeOperation(operation)
        if err != nil {
            log.Errorf("error: %s", err)
            // Handle a process failure by returning an error or time.Duration
        }

        // If your step finishes with data which should be added to override used during the Runtime upgrade,
        // add an extra value to operation.InputCreator, then return the updated version of the Application
        updatedOperation.InputCreator.SetOverrides("component-name", []*gqlschema.ConfigEntryInput{
            {
                Key:   "some.key",
                Value: body.token,
            },
        })

        // Return the updated version of the Application
        return *updatedOperation, 0, nil
    }
    ```

3. Add the step to the [`/cmd/broker/upgrade_cluster.go`](../../cmd/broker/upgrade_cluster.go) or [`/cmd/broker/upgrade_kyma.go`](../../cmd/broker/upgrade_kyma.go) file:

    ```go
    upgradeSteps := []struct {
   		weight   int
   		step     upgrade_kyma.Step
   	}{
   		{
   			weight: 1,
   			step:   upgrade_kyma.NewHelloWorldStep(db.Operations(), &http.Client{}),
   		},
    }
    ```

   </details>
</div>

## Stages

An operation defines stages and steps which represent the work you must do. A stage is a grouping unit for steps. A step is a part of a stage. An operation can consist of multiple stages, and a stage can consist of multiple steps. You group steps in a stage when you have some sensitive data which you don't want to store in database. In such a case you temporarily store the sensitive data in the memory and go through the steps. Once all the steps in a stage are successfully executed, the stage is marked as finished and never repeated again, even if the next one fails. If any steps fail at a given stage, the whole stage is repeated from the beginning.