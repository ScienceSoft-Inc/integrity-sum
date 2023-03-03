//go:build bee2

package bee2

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/ScienceSoft-Inc/integrity-sum/pkg/hasher"
)

var (
	testData       = bytes.NewBufferString("some test data for hashing")
	expectedValues = map[string]string{
		// standart
		"SHA256": "2b55aa83baaad32c386dab48ff3c6df02784406a223e8aec570782c9e7bd851d",
		// bee2 library
		"BEE2": "b6348a4a66b64b5e88419f25dba58fed88f8fd376ee5b92ff783839684c7e328",
	}
)

// Creates temporary file with data in @buf
func createTmpFile(buf *bytes.Buffer) (string, error) {
	f, err := os.CreateTemp("/tmp", "test_")
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = buf.WriteTo(f)
	return f.Name(), err
}

func TestBee2Hasher(t *testing.T) {
	fileName, err := createTmpFile(testData)
	assert.NoError(t, err, "test file creation")
	defer func() {
		os.Remove(fileName)
		fmt.Println("file removed:", fileName)
	}()
	fmt.Println("file created:", fileName)

	log := logrus.New()
	for algName, want := range expectedValues {
		h := hasher.NewFileHasher(algName, log)
		hash, err := h.HashFile(fileName)
		assert.NoError(t, err)
		assert.Equal(t, want, hash, "alg: %v", algName)
	}

	// bee2 with Go Hash interface
	data, err := os.ReadFile(fileName)
	assert.NoError(t, err)

	h := hasher.NewFileHasher("BEE2", log)
	goHasher, ok := h.(*hasher.Hasher)
	assert.True(t, ok)

	hash, err := goHasher.HashData(bytes.NewReader(data))
	assert.NoError(t, err)
	assert.Equal(t, expectedValues["BEE2"], hash)
}
