package hasher

import (
	"encoding/hex"
	"io"
	"log"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const testFileName = "../../.editorconfig"

// TODO: repeat test with the same hasher

// before
// var expectedValues = map[string]string{
// 	"MD5":    "f670b69b3d123fa53e3e1848a0b6bf6b",
// 	"SHA1":   "da39a3ee5e6b4b0d3255bfef95601890afd80709",
// 	"SHA224": "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f",
// 	"SHA384": "38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b",
// 	"SHA512": "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
// 	"SHA256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
// }

// new
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

func TestNewHashSum(t *testing.T) {
	hashAlgos := []string{"MD5", "SHA1", "SHA224", "SHA384", "SHA512", "SHA256"}
	file, err := os.OpenFile(testFileName, os.O_RDONLY, 0)
	if err != nil {
		log.Printf("Error reading file[%s]: %s", testFileName, err)
		t.Fail()
		return
	}
	for _, v := range hashAlgos {
		// hashSummator := NewHashSum(v)
		hashSummator := NewFileHasher(v, logrus.New()).(*Hasher).h

		_, err = io.Copy(hashSummator, file)
		if err != nil {
			log.Printf("Error reading file[%s] with algorithm[%s]: %s", testFileName, v, err)
		}
		res := hex.EncodeToString(hashSummator.Sum(nil))
		if res != expectedValues[v] {
			log.Printf("New hash not corresponding to default value. %s(new hash) != %s(default value)", res, expectedValues[v])
			t.Fail()
			return
		}
	}
}
