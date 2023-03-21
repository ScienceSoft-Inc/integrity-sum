package integritymonitor

import (
	"context"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/walker"
	"github.com/ScienceSoft-Inc/integrity-sum/internal/worker"
)

// DirSnapshot is a snapshot of a directory.
type DirSnapshot struct {
	dirName    string
	fileHashes []worker.FileHash
}

const defaultHashSize = 128

// HashDir calculates file hashes of a given directory
func HashDir(dir, alg string) *DirSnapshot {
	ctx := context.Background()
	log := logrus.StandardLogger()
	fileHachC := worker.WorkersPool(
		runtime.NumCPU(),
		walker.ChanWalkDir(ctx, dir, log),
		worker.NewWorker(ctx, alg, log),
	)
	ds := DirSnapshot{
		dirName:    dir,
		fileHashes: make([]worker.FileHash, 0, defaultHashSize),
	}

	log.Debugf("dir: %s", ds.dirName)
	for v := range fileHachC {
		v.Path = strings.TrimPrefix(v.Path, dir+"/")
		ds.fileHashes = append(ds.fileHashes, v)
		log.Debugf("file hash: %s \t%s", v.Path, v.Hash)
	}
	return &ds
}

// TODO: func-bridge for dir path - local file system against shared processes
// file system to compare stored and calculated hashes
