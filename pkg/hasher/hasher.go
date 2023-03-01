package hasher

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
)

type InitFunc func() hash.Hash

func init() {
	// default algs
	RegisterAlg("MD5", md5.New)
	RegisterAlg("SHA1", sha1.New)
	RegisterAlg("SHA224", sha256.New)
	RegisterAlg("SHA256", sha256.New)
	RegisterAlg("SHA384", sha512.New)
	RegisterAlg("SHA512", sha512.New)
}

var algs = make(map[string]InitFunc)

// Registers init func for @name hasher
func RegisterAlg(name string, f InitFunc) bool {
	algs[name] = f
	fmt.Printf("algorithm %q has been registered\n", name)
	return true
}

// NewHashSum takes a hashing algorithm name as input and returns registered (or
// default, if name is not defined) hasher for this algorithm.
func NewHashSum(algName string) hash.Hash {
	initFunc, ok := algs[algName]
	if !ok {
		// returns default, sha256
		return sha256.New()
	}
	return initFunc()
}
