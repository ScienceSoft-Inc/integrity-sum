package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/integrity-sum/internal/core/services"
	"github.com/integrity-sum/internal/repositories"
	"github.com/integrity-sum/pkg/api"
	logConfig "github.com/integrity-sum/pkg/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"runtime"
)

// Initializes the binding of the flag to a variable that must run before the main() function
func init() {
	fsLog := pflag.NewFlagSet("log", pflag.ContinueOnError)
	fsLog.Int("v", 5, "verbose level")
	fsSum := pflag.NewFlagSet("sum", pflag.ContinueOnError)
	fsSum.Int("count-workers", runtime.NumCPU(), "number of running workers in the workerpool")
	fsSum.String("algorithm", "SHA256", "algorithm MD5, SHA1, SHA224, SHA256, SHA384, SHA512, default: SHA256")
	fsSum.String("dirPath", "/etc/apt", "name of configMap for hasher")
	fsSum.Bool("doHelp", false, "help")
	if err := viper.BindPFlags(fsSum); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	flag.Parse()

	// Initialize config for logger
	logger := logConfig.InitLogger(viper.GetInt("v"))

	// Install context and signal
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	defer func() {
		signal.Stop(sig)
		cancel()
	}()

	switch {
	case viper.GetBool("doHelp"):
		flag.Usage = func() {
			fmt.Fprintf(os.Stderr, "Custom help %s:\nYou can use the following flag:\n", os.Args[0])

			flag.VisitAll(func(f *flag.Flag) {
				fmt.Fprintf(os.Stderr, "  flag -%v \n       %v\n", f.Name, f.Usage)
			})
		}
		flag.Usage()
	case len(viper.GetString("dirPath")) > 0:
		// Initialize repository
		repository := repositories.NewAppRepository(logger)

		// Initialize service
		service := services.NewAppService(repository, viper.GetString("algorithm"), logger)

		jobs := make(chan string)
		results := make(chan *api.HashData)

		go service.WorkerPool(jobs, results)
		go api.SearchFilePath(viper.GetString("dirPath"), jobs, logger)
		for {
			select {
			case hashData, ok := <-results:
				if !ok {
					return
				}
				fmt.Printf("%s %s\n", hashData.Hash, hashData.FileName)
			case <-sig:
				fmt.Println("exit program")
				return
			case <-ctx.Done():
				fmt.Println("program termination after receiving a signal")
				return
			}
		}
	default:
		logger.Println("use the -h flag on the command line to see all the flags in this app")
	}
}
