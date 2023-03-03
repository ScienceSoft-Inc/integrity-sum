//go:build bee2

package bee2

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/ScienceSoft-Inc/integrity-sum/pkg/hasher"
)

const fname = "../../../go.sum"

var expectedValues = map[string]string{
	// standart
	"SHA256": "88370d2c5fc8452a05ad7382c8df5902a76f27639970d059ef44d1c66e3267f2",
	// bee2 library
	// "BEE2": "0c7bc5135b916cf99acee975062f751f", // hid:16
	"BEE2": "8338f17aa18424e1ed3c1e3c5071149505af67ce164d6e81a15c023621fe2b02", // hid:32
}

func TestBee2Hasher(t *testing.T) {
	log := logrus.New()
	absName, err := filepath.Abs(fname)
	assert.NoError(t, err)

	for algName, want := range expectedValues {
		h := hasher.NewFileHasher(algName, log)
		hash, err := h.HashFile(absName)
		assert.NoError(t, err)
		assert.Equal(t, want, hash)
	}

	// bee2 with Go Hash interface
	data, err := os.ReadFile(absName)
	assert.NoError(t, err)

	h := hasher.NewFileHasher("BEE2", log)
	goHasher, ok := h.(*hasher.Hasher)
	assert.True(t, ok)

	hash, err := goHasher.HashData(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, expectedValues["BEE2"], hash)
}
