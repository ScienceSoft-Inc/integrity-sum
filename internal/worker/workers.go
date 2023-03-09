package worker

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/services/filehash"
	"github.com/ScienceSoft-Inc/integrity-sum/pkg/hasher"
)

type HashWorker func(ind int, fileNameC <-chan string, hashC chan<- filehash.FileHash)

func WorkersPool(countWorkers int, fileNameC <-chan string, w HashWorker) <-chan filehash.FileHash {
	hashC := make(chan filehash.FileHash, countWorkers)
	go func() {
		defer close(hashC)
		var wg sync.WaitGroup
		wg.Add(countWorkers)
		for i := 0; i < countWorkers; i++ {
			go func(ind int, wg *sync.WaitGroup) {
				defer wg.Done()
				w(ind, fileNameC, hashC)
			}(i, &wg)
		}
		wg.Wait()
	}()

	return hashC
}

func NewWorker(ctx context.Context, algName string, log *logrus.Logger) HashWorker {
	return func(ind int, fileNameC <-chan string, hashC chan<- filehash.FileHash) {
		h := hasher.NewFileHasher(algName, log)
		for v := range fileNameC {
			select {
			case <-ctx.Done():
				return
			default:
			}

			hash, err := h.HashFile(v)
			if err != nil {
				log.WithError(err).WithField("file", v).Error("calculate hash")
				continue
			}
			hashC <- filehash.FileHash{
				Path: v,
				Hash: hash,
			}
		}
	}
}
