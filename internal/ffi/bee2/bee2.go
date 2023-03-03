//go:build bee2

package bee2

import (
	"hash"

	"github.com/ScienceSoft-Inc/integrity-sum/pkg/hasher"
)

func init() {
	hasher.RegisterAlg("BEE2", NewDefault)
}

type bee2 struct {
	hid      int
	hashSize int
	state    []byte // cumulative alg calculations
	hash     []byte // result

	data []byte // data to process
	n    int    // actual count of bytes written into data
}

type Option func(*bee2)

var (
	_ hash.Hash         = (*bee2)(nil)
	_ hasher.FileHasher = (*bee2)(nil)
)

func NewDefault() hash.Hash {
	// TODO: take hid, hashSize from args
	return New()
}

func New(opts ...Option) hash.Hash {
	objRef := &bee2{
		hid:      HID,
		hashSize: HASHSIZE,
		data:     make([]byte, BLOCKSIZE),
		hash:     make([]byte, HASHSIZE),
		state:    make([]byte, BLOCKSIZE),
	}

	for _, opt := range opts {
		opt(objRef)
	}

	objRef.bashHashStart() // init alg
	return objRef
}

func WithHid(hid int) Option {
	return func(o *bee2) {
		o.hid = hid
	}
}

func WithHashSize(hs int) Option {
	return func(o *bee2) {
		o.hashSize = hs
	}
}

// io.Writer interface.
func (a *bee2) Write(p []byte) (n int, err error) {
	takeFrom, takeTo := 0, BLOCKSIZE

	for takeFrom < len(p) {
		if takeTo > len(p) {
			takeTo = len(p)
		}
		a.n = copy(a.data, p[takeFrom:takeTo])
		takeFrom += a.n
		takeTo = takeFrom + BLOCKSIZE

		a.bashHashStepH()
	}
	n = takeFrom
	return
}

// Hash interface.
func (a *bee2) Sum(in []byte) []byte {
	hash := a.checkSum()
	return append(in, hash...)
}

func (a *bee2) checkSum() []byte {
	a.bashHashStepG()
	return a.hash
}

// Hash interface.
func (a *bee2) Reset() {
	emptyHash := [HASHSIZE]byte{}
	copy(a.hash, emptyHash[:])

	emptyState := [BLOCKSIZE]byte{}
	copy(a.state, emptyState[:])

	a.bashHashStart()
}

// Hash interface.
func (a *bee2) Size() int { return a.hashSize }

// Hash interface.
func (a *bee2) BlockSize() int { return BLOCKSIZE }

// hasher.FileHasher interface
func (a *bee2) HashFile(fileName string) (string, error) {
	return Bee2HashFile(fileName)
}
