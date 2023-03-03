package hasher

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var (
	testData       = bytes.NewBufferString("some test data for hashing")
	expectedValues = map[string]string{
		"MD5":    "7e28b47200542b8a10a949493dbf8d89",
		"SHA1":   "5c2587b147d1674472623579b5f6248bbf196603",
		"SHA224": "9a9d024f73dd10793f201d5cf348cf42f6c9709d070a2a100de54fbe",
		"SHA384": "271afdf2c8143d1a02d646a67d9767a53a0982820b6839d218e62c78bf7299e58841fe5c943606edcc4d1d7b76a58bc4",
		"SHA512": "60092d6bb71c0b0e7426d97c5291e8548eb776358b1d524da6b2f5af41fa2d255e812c82c2352276f55336ae2aba6c8784ccd1f9099b4e5cb3bb82bf3f7ae357",
		"SHA256": "2b55aa83baaad32c386dab48ff3c6df02784406a223e8aec570782c9e7bd851d",
	}
)

func TestHasher(t *testing.T) {
	fileName, err := createTmpFile(testData)
	assert.NoError(t, err, "test file creation")
	defer func() {
		os.Remove(fileName)
		fmt.Println("file removed:", fileName)
	}()
	fmt.Println("file created:", fileName)

	testHashAlgs(t, fileName)
	testRepeated(t, fileName)
}

func testHashAlgs(t *testing.T, fileName string) {
	for alg := range expectedValues {
		hash, err := NewFileHasher(alg, logrus.New()).HashFile(fileName)
		assert.NoError(t, err)
		assert.Equal(t, expectedValues[alg], hash, "alg: %s", alg)
	}
}

func testRepeated(t *testing.T, fileName string) {
	alg := "SHA256"
	hasher := NewFileHasher(alg, logrus.New())
	// 1
	hash, err := hasher.HashFile(fileName)
	assert.NoError(t, err)
	assert.Equal(t, expectedValues[alg], hash, "alg: %s", alg)
	// 2
	hash, err = hasher.HashFile(fileName)
	assert.NoError(t, err)
	assert.Equal(t, expectedValues[alg], hash, "alg: %s", alg)
}

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
