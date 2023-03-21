package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/integritymonitor"
)

/*
  Tool for creating snapshots of a file system.

  It calculates file hashes of a given directory and store them as a file for
  further usage.

  It reuses the code of the main repo and particularly the setupIntegrity()
  function.

  Example of usage:
  ./snapshot --dir="/tmp/dir1,/tmp/dir2" --dir dir3 --out /tmp/snapshot.txt
*/

func main() {
	initConfig()
	initLog()

	dirs := viper.GetStringSlice("dir")
	logrus.Debugf("len(dirs): %v", len(dirs))
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logrus.Fatalf("dir %s does not exist", dir)
		}

		integritymonitor.HashDir(dir, viper.GetString("algorithm"))
	}

}

func initConfig() {
	// pflag.String("verbose", "info", "verbose level")
	// pflag.String("algorithm", "SHA256", "hashing algorithm for calculating hashes")
	pflag.StringSlice("dir", []string{}, "path to dir for which snapshot will be created, example: --dir=\"tmp,bin\" --dir vendor (result: [tmp bin vendor])")
	pflag.String("out", "out.txt", "output file name")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
}

func initLog() {
	lvl, err := logrus.ParseLevel(viper.GetString("verbose"))
	if err != nil {
		logrus.WithError(err).Error("Failed to parse log level")
	}
	logrus.SetLevel(lvl)
}
