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

const testFileName = "../../../.editorconfig"

var expectedValues = map[string]string{
	// standart
	"SHA256": "5a56cf93d0987654cd2cad1b6616e1f413b0984c59e56470f450176246e42e47",
	// bee2 library
	"BEE2": "473c1b38fbd2f1e6480776370c56a317a4702e2742c28b27679cba013678529f",
}

func TestBee2Hasher(t *testing.T) {
	log := logrus.New()
	absName, err := filepath.Abs(testFileName)
	assert.NoError(t, err)

	for algName, want := range expectedValues {
		h := hasher.NewFileHasher(algName, log)
		hash, err := h.HashFile(absName)
		assert.NoError(t, err)
		assert.Equal(t, want, hash, "alg: %v", algName)
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
