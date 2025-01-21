package command

import (
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "skr-tester",
		Short:        "SKR tester",
		Long:         "SKR tester",
		SilenceUsage: true,
	}
	cmd.PersistentFlags().BoolP("help", "h", false, "Option that displays help for the CLI.")
	cmd.AddCommand(
		NewProvisionCmd(),
		NewDeprovisionCmd(),
		NewCheckOperationCommand(),
		NewUpdateCommand(),
		NewAsertCmd(),
		NewBindingCmd(),
	)

	return cmd
}
