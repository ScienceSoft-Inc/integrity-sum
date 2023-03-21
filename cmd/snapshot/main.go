package main

import (
	"encoding/json"
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
  ./snapshot --root-fs="bin/docker-fs" --verbose=debug --dir "/app,/bin" --dir "/dir3" --out "/bin/snapshot.txt"
*/

func main() {
	initConfig()
	initLog()

	if err := snapDirs(); err != nil {
		logrus.WithError(err).Error("Failed to create output file")
	}
}

func initConfig() {
	// pflag.String("verbose", "info", "verbose level")
	// pflag.String("algorithm", "SHA256", "hashing algorithm for calculating hashes")
	pflag.StringSlice("dir", []string{}, "path to dir for which snapshot will be created, example: --dir=\"tmp,bin\" --dir vendor (result: [tmp bin vendor])")
	pflag.String("root-fs", "./", "path to docker image root filesystem")
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

func snapDirs() error {
	rootFs := viper.GetString("root-fs")
	// logrus.Debugf("root-fs: %v", rootFs)
	dirs := viper.GetStringSlice("dir")
	// logrus.Debugf("len(dirs): %v", len(dirs))

	outFileName := viper.GetString("out")
	// logrus.Debugf("out: %v", outFileName)
	file, err := os.Create(outFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, v := range dirs {
		dir := rootFs + v
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			logrus.Fatalf("dir %s does not exist", dir)
		}

		snapshot := integritymonitor.HashDir(rootFs, v, viper.GetString("algorithm"))
		// snapJson := snapshot.ToJson()
		bs, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		file.Write(bs)
	}
	return nil
}
