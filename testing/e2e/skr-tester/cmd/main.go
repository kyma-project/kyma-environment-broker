package main

import (
	"fmt"
	"os"
	"os/signal"
	"skr-tester/pkg/command"
	"syscall"
)

func main() {
	setupCloseHandler()
	cmd := command.New()

	err := cmd.Execute()
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
