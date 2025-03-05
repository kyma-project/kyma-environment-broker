package main

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	rootCmd = &cobra.Command{
		Use:     "hap",
		Short:   "A tool for parsing and validation of HAP rules",
		Version: "v0.0.12",
		Long:    ``,
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
}
