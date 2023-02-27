package main

import (
	"context"
	"flag"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/graceful"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/initialize"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/logger"
)

func main() {
	// Install config
	initConfig()

	// Install logger
	logger := logger.Init(viper.GetString("verbose"))

	// Install migration
	DBMigration(logger)

	// Run Application with graceful shutdown context
	graceful.Execute(context.Background(), logger, func(ctx context.Context) {
		initialize.Initialize(ctx, logger)
	})
}

func initConfig() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.AutomaticEnv()
}
