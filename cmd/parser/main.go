package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
)

var gitCommit string

var (
	rootCmd = &cobra.Command{
		Use:     "hap",
		Short:   "Check HAP rules",
		Version: gitCommit,
		Long:    "A tool for parsing and matching data using HAP rules",
	}
)

func main() {
	setupCloseHandler()
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
