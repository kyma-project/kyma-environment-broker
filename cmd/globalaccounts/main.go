package main

import (
	"context"

	"github.com/kyma-project/kyma-environment-broker/internal/globalaccounts"
	"github.com/vrischmann/envconfig"
)

func main() {
	var cfg globalaccounts.Config
	err := envconfig.InitWithPrefix(&cfg, "GLOBALACCOUNTS")
	if err != nil {
		panic(err.Error())
	}
	ctx := context.Background()
	globalaccounts.Run(ctx, cfg)
}
