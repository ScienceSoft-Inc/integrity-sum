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
	dirName    string            `json:"dir"`
	fileHashes []worker.FileHash `json:"hashes"`
}

const defaultHashSize = 128

// HashDir calculates file hashes of a given directory
func HashDir(rootFs, pathToMonitor, alg string) *DirSnapshot {

	ctx := context.Background()
	log := logrus.StandardLogger()
	dir := rootFs + pathToMonitor
	fileHachC := worker.WorkersPool(
		runtime.NumCPU(),
		walker.ChanWalkDir(ctx, dir, log),
		worker.NewWorker(ctx, alg, log),
	)
	ds := DirSnapshot{
		dirName:    pathToMonitor,
		fileHashes: make([]worker.FileHash, 0, defaultHashSize),
	}

	// log.Debugf("dir: %s", ds.dirName)
	for v := range fileHachC {
		v.Path = strings.TrimPrefix(v.Path, rootFs)
		ds.fileHashes = append(ds.fileHashes, v)
		// log.Debugf("file hash: %s \t%s", v.Path, v.Hash)
	}

	return &ds
}

// Snapshot contains snapshots of selected directories.
type Snapshot struct {
	Dirs []DirSnapshot `json:"dirs"`
}

// func (s *Snapshot) ToJson() ([]byte, error) {
// 	bs, err := json.Marshal(s)
// }

// TODO: func-bridge for dir path - local file system against shared processes
// file system to compare stored and calculated hashes
