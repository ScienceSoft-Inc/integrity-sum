package hasher

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/ScienceSoft-Inc/integrity-sum/internal/ffi/bee2"
)

// NewHashSum takes a hashing algorithm as input and returns a hash sum with other data or an error
func NewHashSum(alg string) hash.Hash {

	switch alg {
	case "MD5":
		return md5.New()
	case "SHA1":
		return sha1.New()
	case "SHA224":
		return sha256.New224()
	case "SHA384":
		return sha512.New384()
	case "SHA512":
		return sha512.New()
	case "BEE2":
		return bee2.New()
	case "SHA256":
		fallthrough
	default:
		return sha256.New()
	}
}
