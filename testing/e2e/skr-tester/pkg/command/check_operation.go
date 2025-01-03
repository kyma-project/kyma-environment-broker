package command

import (
	"errors"
	"fmt"
	broker "skr-tester/pkg/broker"
	"skr-tester/pkg/logger"
	"time"

	"github.com/spf13/cobra"
)

type CheckOperationCommand struct {
	cobraCmd    *cobra.Command
	log         logger.Logger
	instanceID  string
	operationID string
	timeout     time.Duration
	interval    time.Duration
}

func NewCheckOperationCommand() *cobra.Command {
	cmd := CheckOperationCommand{}
	cobraCmd := &cobra.Command{
		Use:     "operation",
		Aliases: []string{"o"},
		Short:   "Waits for operation to finish",
		Long:    "Waits for operation to finish",
		Example: "skr-tester operation -i instanceID -op operationID                            Checks the operation status.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "InstanceID of the specific instance.")
	cobraCmd.Flags().StringVarP(&cmd.operationID, "operationID", "o", "", "OperationID of the specific operation.")
	cobraCmd.Flags().DurationVarP(&cmd.timeout, "timeout", "t", 40*time.Minute, "Timeout for the operation to finish.")
	cobraCmd.Flags().DurationVarP(&cmd.interval, "interval", "n", 1*time.Minute, "Interval between operation checks.")

	return cobraCmd
}

func (cmd *CheckOperationCommand) Run() error {
	cmd.log = logger.New()
	brokerClient := broker.NewBrokerClient(broker.NewBrokerConfig())
	var state string
	err := wait(func() (bool, error) {
		var err error
		resp, err := brokerClient.GetOperation(cmd.instanceID, cmd.operationID)
		if err != nil {
			return false, err
		}
		var ok bool
		state, ok = resp["state"].(string)
		if !ok {
			return false, errors.New("state field not found in operation response")
		}
		return state == "succeeded" || state == "failed", nil
	}, cmd.timeout, cmd.interval)
	if err != nil {
		return err

	}
	if state != "succeeded" {
		return errors.New(fmt.Sprintf("error thrown by ensureOperationSucceeded: operation didn't succeed in time. Final state: %s", state))
	}

	fmt.Printf("Operation %s finished with state %s\n", cmd.operationID, state)
	return nil
}

func (cmd *CheckOperationCommand) Validate() error {
	if cmd.instanceID != "" && cmd.operationID != "" {
		return nil
	} else {
		return errors.New("both of the following options have to be specified: instanceID, operationID")
	}
}

func wait(condition func() (bool, error), timeout, interval time.Duration) error {
	done, err := condition()
	if err != nil {
		return err
	}
	if done {
		return nil
	}
	timeoutCh := time.After(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-timeoutCh:
			return errors.New("timeout reached")
		case <-ticker.C:
			done, err := condition()
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}
