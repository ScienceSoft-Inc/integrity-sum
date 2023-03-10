package hasher

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Expands default Go Hash interface
type FileHasher interface {
	HashFile(fileName string) (string, error)
}

type Hasher struct {
	h   hash.Hash
	log *logrus.Logger
}

type InitFunc func() hash.Hash

var algs = make(map[string]InitFunc)

func init() {
	// default algs
	RegisterAlg("MD5", md5.New)
	RegisterAlg("SHA1", sha1.New)
	RegisterAlg("SHA224", sha256.New224)
	RegisterAlg("SHA256", sha256.New)
	RegisterAlg("SHA384", sha512.New384)
	RegisterAlg("SHA512", sha512.New)
}

// HashFile calculates hash for file
func (fh *Hasher) HashFile(fullFileName string) (string, error) {
	if ownFileHasher, ok := fh.h.(FileHasher); ok {
		return ownFileHasher.HashFile(fullFileName)
	}

	data, err := os.ReadFile(fullFileName)
	if err != nil {
		fh.log.WithError(err).Errorf("can not read from file %q", fullFileName)
		return "", err
	}
	return fh.HashData(bytes.NewReader(data))
}

// HashData calculates hash for a data represented with @r.
// Uses default Go Hash interface.
func (fh *Hasher) HashData(r *bytes.Reader) (string, error) {
	fh.h.Reset()
	_, err := io.Copy(fh.h, r)
	if err != nil {
		fh.log.WithError(err).Error("io.Copy()")
		return "", err
	}
	return hex.EncodeToString(fh.h.Sum(nil)), nil
}

// Registers init func for @name algorithm
func RegisterAlg(name string, f InitFunc) bool {
	algs[name] = f
	fmt.Printf("algorithm %q has been registered\n", name)
	return true
}

// newHasherInstance takes a hashing algorithm name as input and returns
// registered (or default, if name is not recognized) hasher for this algorithm.
func newHasherInstance(algName string) hash.Hash {
	initFunc, ok := algs[algName]
	if !ok {
		// returns default, sha256
		return sha256.New()
	}
	return initFunc()
	// TODO: log actual name of selected alg. Or return true/false signal
}

// Returns new FileHasher instance
func NewFileHasher(algName string, log *logrus.Logger) FileHasher {
	return &Hasher{
		h:   newHasherInstance(algName),
		log: log,
	}
}
