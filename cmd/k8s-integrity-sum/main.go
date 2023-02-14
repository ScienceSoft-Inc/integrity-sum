package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/integrity-sum/internal/initialize"
	logConfig "github.com/integrity-sum/pkg/logger"
)

func init() {
	fsLog := pflag.NewFlagSet("log", pflag.ContinueOnError)
	fsLog.Int("v", 5, "verbose level")
	if err := viper.BindPFlags(fsLog); err != nil {
		fmt.Printf("error binding flags: %v", err)
		os.Exit(2)
		return
	}
}

func main() {
	initConfig()
	logger := logConfig.InitLogger(viper.GetInt("v"))

	DBMigration(logger)

	// Handling shutdown signals
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		signal.Stop(sig)
		cancel()
	}()

	// Initialize program
	initialize.Initialize(ctx, logger, sig)
}

func initConfig() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.AutomaticEnv()
}
