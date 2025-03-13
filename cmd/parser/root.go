package main

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	rootCmd = &cobra.Command{
		Use:     "hap",
		Short:   "A tool for parsing and validation of HAP rules",
		Version: "0.0.13",
		Long:    ``,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
}
