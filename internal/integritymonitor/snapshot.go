package integritymonitor

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/walker"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/worker"
)

const DefaultHashSize = 128

// HashDir calculates file hashes of a given directory
func HashDir(rootFs, pathToMonitor, alg string) []worker.FileHash {
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("scan-dir-timeout"))
	defer cancel()
	log := logrus.StandardLogger()
	dir := rootFs + pathToMonitor
	fileHachC := worker.WorkersPool(
		runtime.NumCPU(),
		walker.ChanWalkDir(ctx, dir, log),
		worker.NewWorker(ctx, alg, log),
	)

	hashes := make([]worker.FileHash, 0, DefaultHashSize)
	for v := range fileHachC {
		v.Path = strings.TrimPrefix(v.Path, rootFs)
		hashes = append(hashes, v)
	}
	return hashes
}

// CalculateAndWriteHashes calculates file hashes of a given directory and store
// them as a file for further usage.
func CalculateAndWriteHashes() error {
	rootFs := viper.GetString("root-fs")
	dirs := viper.GetStringSlice("dir")

	file, err := os.Create(viper.GetString("out"))
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		if err != nil {
			os.Remove(file.Name())
		}
	}()

	hashes := make([]worker.FileHash, 0, DefaultHashSize*len(dirs))
	for _, v := range dirs {
		dir := rootFs + v
		if _, err = os.Stat(dir); os.IsNotExist(err) {
			logrus.Errorf("dir %s does not exist", dir)
			return err
		}
		hashes = append(hashes, HashDir(rootFs, v, viper.GetString("algorithm"))...)
	}

	err = writeHashesJson(file, hashes)
	return err
}

func writeHashesJson(file *os.File, hashes []worker.FileHash) error {
	bs, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		logrus.Errorf("failed to marshal snapshot: %v", err)
		return err
	}
	if _, err = file.Write(bs); err != nil {
		logrus.Errorf("failed to write snapshot: %v", err)
		return err
	}
	return nil

	/*
		[
		  {
		    "path": "/app/db/migrations/000001_init.down.sql",
		    "hash": "39c1fa1a6fed5662a372df6b9c717e526cb7e2f4adcacd5a7224cb9ab62730cd"
		  },
		  {
		    "path": "/app/db/migrations/000001_init.up.sql",
		    "hash": "11539904589278c0fea68b1aca1e86490d796392af1f437dfe2ea0c8ec469cd6"
		  },
		  {
		    "path": "/app/integritySum",
		    "hash": "d4b7246928b0420ea5143a7db9cbd63db5195c14caa7ed2568b883eefad02731"
		  },
		  {
		    "path": "/bin/busybox",
		    "hash": "36d96947f81bee3a5e1d436a333a52209f051bb3556028352d4273a748e2d136"
		  }
		]
	*/
}
