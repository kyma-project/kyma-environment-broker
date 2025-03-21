package command

import (
	"fmt"

	"skr-tester/pkg/kcp"
	"skr-tester/pkg/logger"

	"github.com/spf13/cobra"
)

type EventsCommand struct {
	cobraCmd   *cobra.Command
	log        logger.Logger
	instanceID string
}

func NewEventsCmd() *cobra.Command {
	cmd := EventsCommand{}
	cobraCmd := &cobra.Command{
		Use:     "events",
		Aliases: []string{"e"},
		Short:   "Gets events for the instance",
		Long:    "Gets events for the instance",
		Example: "	skr-tester events -i instanceID                            Gets events for the instance.",

		PreRunE: func(_ *cobra.Command, _ []string) error { return cmd.Validate() },
		RunE:    func(_ *cobra.Command, _ []string) error { return cmd.Run() },
	}
	cmd.cobraCmd = cobraCmd

	cobraCmd.Flags().StringVarP(&cmd.instanceID, "instanceID", "i", "", "Instance ID of the specific instance.")

	return cobraCmd
}

func (cmd *EventsCommand) Run() error {
	cmd.log = logger.New()
	kcpClient, err := kcp.NewKCPClient()
	if err != nil {
		return fmt.Errorf("failed to create KCP client: %v", err)
	}
	events, err := kcpClient.GetEvents(cmd.instanceID)
	if err != nil {
		return fmt.Errorf("failed to get events: %v", err)
	}
	fmt.Println(events)

	return nil
}

func (cmd *EventsCommand) Validate() error {
	if cmd.instanceID != "" {
		return nil
	} else {
		return fmt.Errorf("at least one of the following options have to be specified: instanceID")
	}
}
