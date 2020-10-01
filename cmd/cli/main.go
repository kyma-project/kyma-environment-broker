package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/logger"
	"github.com/kyma-project/control-plane/components/kyma-environment-broker/cmd/cli/skr"
)

func main() {
	setupCloseHandler()
	log := logger.New()
	command := skr.NewCmd(log)

	err := command.Execute()
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
