package hasher

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const testFileName = "../../.editorconfig"

var expectedValues = map[string]string{
	// $ md5sum .editorconfig
	// f670b69b3d123fa53e3e1848a0b6bf6b  .editorconfig
	"MD5": "f670b69b3d123fa53e3e1848a0b6bf6b",

	"SHA1": "40ed457cdd863b52153246992298e4c10b4c7833",

	// $ sha224sum .editorconfig
	// 91aab9267c12bf42be3ae87f15afcb1adc52e6301076f5094c843b78  .editorconfig
	"SHA224": "91aab9267c12bf42be3ae87f15afcb1adc52e6301076f5094c843b78",

	"SHA384": "80f04572e3078da6d775adc7de9cea8af9c141fd12bd232fc22c43bc27f67559c34d20b666514e7bdfa68cd11c2e45e7",

	// $ sha512sum .editorconfig
	// dda4a0dd082392a0447afe99dc67b4b39170dd84731986015b047e2e70237232a276e5899aaaca544103f34b61ddfbb91e0ed57b76afe80d2bf65e982a8e7724  .editorconfig
	"SHA512": "dda4a0dd082392a0447afe99dc67b4b39170dd84731986015b047e2e70237232a276e5899aaaca544103f34b61ddfbb91e0ed57b76afe80d2bf65e982a8e7724",

	// $ sha256sum .editorconfig
	// 5a56cf93d0987654cd2cad1b6616e1f413b0984c59e56470f450176246e42e47  .editorconfig
	"SHA256": "5a56cf93d0987654cd2cad1b6616e1f413b0984c59e56470f450176246e42e47",
}

func TestHashAlgs(t *testing.T) {
	for alg := range expectedValues {
		hash, err := NewFileHasher(alg, logrus.New()).HashFile(testFileName)
		assert.NoError(t, err)
		assert.Equal(t, expectedValues[alg], hash, "alg: %s", alg)
	}
}

func TestRepeated(t *testing.T) {
	alg := "SHA256"
	hasher := NewFileHasher(alg, logrus.New())
	// 1
	hash, err := hasher.HashFile(testFileName)
	assert.NoError(t, err)
	assert.Equal(t, expectedValues[alg], hash, "alg: %s", alg)
	// 2
	hash, err = hasher.HashFile(testFileName)
	assert.NoError(t, err)
	assert.Equal(t, expectedValues[alg], hash, "alg: %s", alg)
}
