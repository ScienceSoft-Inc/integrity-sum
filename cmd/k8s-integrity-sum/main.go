package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	_ "github.com/ScienceSoft-Inc/integrity-sum/internal/configs"
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
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/minio"
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

	_, err = minio.NewStorage(log)
	if err != nil {
		log.Fatalf("failed connect to minio storage: %w", err)
	}

	k8s.InitKubeData()
	kubeClient := k8s.NewKubeService(log)
	err = kubeClient.Connect()
	if err != nil {
		log.Fatalf("failed connect to kubernetes: %w", err)
	}

	deploymentData, err := kubeClient.GetDataFromDeployment()
	if err != nil {
		log.Fatalf("failed get deployment data: %w", err)
	}

	// Create alert sender
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
		syslogSender := syslogclient.New(
			log,
			viper.GetString("syslog-proto"),
			addr,
			syslogclient.DefaultPriority,
			fmt.Sprintf("%s.%s", deploymentData.NameDeployment, deploymentData.NameSpace),
			common.AppId)
		alerts.Register(syslogSender)
		log.Info("notification to syslog enabled")
	}

	optsMap, err := integritymonitor.ParseMonitoringOpts(viper.GetString("monitoring-options"))
	if err != nil {
		log.WithError(err).Fatal("cannot parse monitoring options")
	}

	// Run Application with graceful shutdown context
	graceful.Execute(context.Background(), log, func(ctx context.Context) {
		hbAlert := alerts.New("health check", alerts.HeartbeatEvent, "", common.AppId)
		alerts.Heartbeat(ctx, log, hbAlert)

		err := runCheckIntegrity(ctx, log, optsMap, deploymentData, kubeClient)
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

func runCheckIntegrity(ctx context.Context,
	log *logrus.Logger,
	optsMap map[string][]string,
	deploymentData *k8s.DeploymentData,
	kubeClient *k8s.KubeClient) error {

	var err error
	t := time.NewTicker(viper.GetDuration("duration-time"))
	for range t.C {
		for proc, paths := range optsMap {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				log.Info("running a next check loop..")
			}

			err = integritymonitor.CheckIntegrity(ctx, log, proc, paths, deploymentData, kubeClient)
			if err != nil {
				log.WithError(err).Error("failed check integrity")
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
