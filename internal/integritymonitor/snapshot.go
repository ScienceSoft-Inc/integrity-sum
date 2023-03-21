package integritymonitor

import (
	"context"
	"os"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/walker"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/worker"
)

const DefaultHashSize = 128

// CalculateAndWriteHashes calculates file hashes of a given directory and store
// them as a file for further usage.
func CalculateAndWriteHashes() error {
	rootPath := viper.GetString("root-fs") + "/"
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
		dir := rootPath + v
		if _, err = os.Stat(dir); os.IsNotExist(err) {
			logrus.Errorf("dir %s does not exist", dir)
			return err
		}
		hashes = append(hashes, HashDir(rootPath, v, viper.GetString("algorithm"))...)
	}

	err = writeAsPlainText(file, hashes)
	return err
}

// HashDir calculates file hashes of a given directory
func HashDir(rootPath, pathToMonitor, alg string) []worker.FileHash {
	ctx, cancel := context.WithTimeout(context.Background(), viper.GetDuration("scan-dir-timeout"))
	defer cancel()
	log := logrus.StandardLogger()
	fileHachC := worker.WorkersPool(
		runtime.NumCPU(),
		walker.ChanWalkDir(ctx, rootPath+pathToMonitor, log),
		worker.NewWorker(ctx, alg, log),
	)

	hashes := make([]worker.FileHash, 0, DefaultHashSize)
	for v := range fileHachC {
		v.Path = strings.TrimPrefix(v.Path, rootPath)
		hashes = append(hashes, v)
	}
	return hashes
}

func writeAsPlainText(file *os.File, hashes []worker.FileHash) error {
	separator := "  "
	for _, v := range hashes {
		_, err := file.WriteString(v.Hash + separator + v.Path + "\n")
		if err != nil {
			logrus.Errorf("failed to write hashes: %v", err)
			return err
		}
	}
	return nil
}
