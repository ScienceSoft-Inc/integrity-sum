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
	"strings"

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
	RegisterAlg("md5", md5.New)
	RegisterAlg("sha1", sha1.New)
	RegisterAlg("sha224", sha256.New224)
	RegisterAlg("sha256", sha256.New)
	RegisterAlg("sha384", sha512.New384)
	RegisterAlg("sha512", sha512.New)
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

// Registers init func for @algName algorithm
func RegisterAlg(algName string, f InitFunc) bool {
	algName = strings.ToLower(algName)
	algs[algName] = f
	fmt.Printf("algorithm %q has been registered\n", algName)
	return true
}

// newHasherInstance takes a hashing algorithm name as input and returns
// registered (or default, if name is not recognized) hasher for this algorithm.
func newHasherInstance(algName string) hash.Hash {
	algName = strings.ToLower(algName)
	initFunc, ok := algs[algName]
	if !ok {
		// returns default, sha256
		logrus.Debugf("algorithm %q is not registered, sha256 will be used", algName)
		return sha256.New()
	}
	logrus.Debugf("algorithm %q has been selected", algName)
	return initFunc()
}

// Returns new FileHasher instance
func NewFileHasher(algName string, log *logrus.Logger) FileHasher {
	return &Hasher{
		h:   newHasherInstance(algName),
		log: log,
	}
}
