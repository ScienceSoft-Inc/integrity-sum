package main

import (
	"time"

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
  ./snapshot --root-fs="bin/docker-fs" --verbose=debug --dir "/app,/bin" --out "bin/snapshot.txt"

  Exporting docker image filesystem.
  The code below will export the filesystem of the docker image "integrity:latest into the "./bin/docker-fs/":
  cid=$(docker create integrity:latest) && docker export $cid | tar -xC ./bin/docker-fs/ && docker rm $cid
*/

func main() {
	initConfig()
	initLog()

	if err := integritymonitor.CalculateAndWriteHashes(); err != nil {
		logrus.WithError(err).Error("failed to create output file")
	}
}

func initConfig() {
	pflag.StringSlice("dir", []string{}, "path to dir for which snapshot will be created, example: --dir=\"tmp,bin\" --dir vendor (result: [tmp bin vendor])")
	pflag.String("root-fs", "./", "path to docker image root filesystem")
	pflag.String("out", "out.txt", "output file name")
	pflag.Duration("scan-dir-timeout", 30*time.Second, "timeout for scanning directory while creating hashes")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
}

func initLog() {
	vLevel := viper.GetString("verbose")
	lvl, err := logrus.ParseLevel(vLevel)
	if err != nil {
		logrus.WithError(err).WithField("verbose", vLevel).Error("failed to parse log level")
		logrus.SetLevel(logrus.InfoLevel)
		return
	}
	logrus.SetLevel(lvl)
}
