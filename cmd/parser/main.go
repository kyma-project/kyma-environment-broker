package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var gitCommit string
var rootCmd *cobra.Command

func main() {
	setupCloseHandler()

	rootCmd = &cobra.Command{
		Use:           "hap",
		Short:         "A tool for parsing and validation of HAP rules",
		Version:       gitCommit,
		Long:          ``,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	rootCmd.AddCommand(NewParseCmd())

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func setupCloseHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-c
		fmt.Printf("\r- Signal '%v' received from Terminal. Exiting...\n ", sig)
		os.Exit(0)
	}()
}
