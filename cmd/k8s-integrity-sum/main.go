package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"

	_ "github.com/ScienceSoft-Inc/integrity-sum/internal/configs"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/data"
	_ "github.com/ScienceSoft-Inc/integrity-sum/internal/ffi/bee2"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/integritymonitor"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/logger"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/utils/graceful"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts/splunk"
	syslogclient "github.com/ScienceSoft-Inc/integrity-sum/pkg/alerts/syslog"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/common"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/health"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/k8s"
	_ "github.com/ScienceSoft-Inc/integrity-sum/pkg/minio"
)

func main() {
	// Install config
	initConfig()

	// Install logger
	log := logger.Init(viper.GetString("verbose"))

	// Set app health status to healthy
	h := health.New(fmt.Sprintf("/tmp/%s", common.AppId))
	err := h.Set()
	if err != nil {
		log.Fatalf("cannot create health file")
	}
	defer h.Reset()

	// Install migration
	DBMigration(log)

	// DB connect
	if _, err := data.Open(log); err != nil {
		log.Fatalf("failed connect to database: %w", err)
	}

	// 	// Create alert sender
	if viper.GetBool("splunk-enabled") {
		splunkUrl := viper.GetString("splunk-url")
		splunkToken := viper.GetString("splunk-token")
		splunkInsecureSkipVerify := viper.GetBool("splunk-insecure-skip-verify")
		if len(splunkUrl) > 0 && len(splunkToken) > 0 {
			alertsSender := splunk.New(log, splunkUrl, splunkToken, splunkInsecureSkipVerify)
			alerts.Register(alertsSender)
		} else {
			log.Info("splunk URL or Token is missed splunk support disabled")
		}
	}

	if viper.GetBool("syslog-enabled") {
		addr := fmt.Sprintf("%s:%d", viper.GetString("syslog-host"), viper.GetInt("syslog-port"))
		syslogSender, err := syslogclient.New(log, viper.GetString("syslog-proto"), addr, syslogclient.DefaultPriority)
		if err != nil {
			log.Fatalf("cannnot initialize sysylog client %v", err)
		}
		alerts.Register(syslogSender)
		log.Info("notification to syslog enabled")

		defer syslogSender.Close()
	}

	kubeClient := k8s.NewKubeService(log)
	err = kubeClient.Connect()
	if err != nil {
		log.Fatalf("failed connect to kubernetes: %w", err)
	}
	kubeData, err := kubeClient.GetKubeData()
	if err != nil {
		log.Fatalf("failed get kube data: %w", err)
	}
	deploymentData, err := kubeClient.GetDataFromDeployment(kubeData)
	if err != nil {
		log.Fatalf("failed get deployment data: %w", err)
	}

	optsMap, err := integritymonitor.ParseMonitoringOpts(viper.GetString("monitoring-options"))
	if err != nil {
		log.WithError(err).Fatal("cannot parse monitoring options")
	}

	// Run Application with graceful shutdown context
	graceful.Execute(context.Background(), log, func(ctx context.Context) {
		if err = setupIntegrity(ctx, log, deploymentData, optsMap); err != nil {
			log.WithError(err).Fatal("failed to setup integrity")
			return
		}

		err := runCheckIntegrity(ctx, log, optsMap, kubeData, deploymentData, kubeClient)
		if err == context.Canceled {
			log.Info("execution cancelled")
			return
		}
		if err != nil {
			log.WithError(err).Error("monitor execution aborted")
			return
		}
	})
}

func setupIntegrity(ctx context.Context, log *logrus.Logger, deploymentData *k8s.DeploymentData, optsMap map[string][]string) error {
	g := errgroup.Group{}
	for pName, pPaths := range optsMap {
		pName := pName
		pPaths := pPaths
		g.Go(func() error {
			var err error
			processPaths := make([]string, len(pPaths))
			for i, p := range pPaths {
				processPaths[i], err = integritymonitor.GetProcessPath(pName, p)
				if err != nil {
					return err
				}

			}
			if err := integritymonitor.SetupIntegrity(ctx, processPaths, log, deploymentData); err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func runCheckIntegrity(ctx context.Context,
	log *logrus.Logger,
	optsMap map[string][]string,
	kubeData *k8s.KubeData,
	deploymentData *k8s.DeploymentData,
	kubeClient *k8s.KubeClient) error {

	var err error

	t := time.NewTicker(viper.GetDuration("duration-time"))
	for range t.C {
		for proc, paths := range optsMap {
			processPaths := make([]string, len(paths))
			for i, p := range paths {
				processPaths[i], err = integritymonitor.GetProcessPath(proc, p)
				if err != nil {
					log.WithError(err).Error("failed build process path")
					return err
				}

			}
			err = integritymonitor.CheckIntegrity(ctx, log, processPaths, kubeData, deploymentData, kubeClient)
			if err != nil {
				log.WithError(err).Error("failed check integrity")
				return err
			}
		}
	}

	return nil
}

func initConfig() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	viper.AutomaticEnv()
}
