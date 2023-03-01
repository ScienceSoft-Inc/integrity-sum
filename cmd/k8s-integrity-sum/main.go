package main

import (
	"context"
	"flag"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	_ "github.com/ScienceSoft-Inc/integrity-sum/internal/ffi/bee2" // bee2 registration
	"github.com/ScienceSoft-Inc/integrity-sum/internal/graceful"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/initialize"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/logger"
)

func main() {
	// Install config
	initConfig()

	// Install logger
	log := logger.Init(viper.GetString("verbose"))

	// Install migration
	DBMigration(log)

	// Run Application with graceful shutdown context
	graceful.Execute(context.Background(), log, func(ctx context.Context) {
		initialize.Initialize(ctx, log)
	})
}

func initConfig() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.AutomaticEnv()
}
