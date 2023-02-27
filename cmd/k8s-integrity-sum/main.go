package main

import (
	"context"
	"database/sql"
	"flag"
	"path/filepath"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/core/services"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/ffi/bee2"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/graceful"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/initialize"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/logger"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/repositories"
)

func main() {
	// Install config
	initConfig()

	// Install logger
	logger := logger.Init(viper.GetString("verbose"))

	// Only for testing bee2 alg. Do not merge
	// go run ./cmd/k8s-integrity-sum
	{
		fn := "./go.sum"
		p, err := filepath.Abs(fn)
		if err != nil {
			logger.Fatalf("filepath.Abs(fn): %v", err)
		}
		logger.Infof("- whole file hash : %v: %v", filepath.Base(p), bee2.Bee2HashFile(p, logger))

		// hasher interface
		repository := repositories.NewAppRepository(logger, new(sql.DB))
		hs := services.NewHashService(repository, "BEE2", logger)
		hd, err := hs.CreateHash(fn)
		if err != nil {
			logger.Fatalf("CreateHash(): %v", err)
		}
		logger.Infof("- hasher interface: %v: %v", hd.FileName, hd.Hash)
		return
	}

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
