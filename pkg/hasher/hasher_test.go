package hasher

import (
	"encoding/hex"
	"io"
	"log"
	"os"
	"testing"
)

const fp = "/Users/isemenov/go/src/integrity-sum/.editorconfig"

var defuletValues = map[string]string{
	"MD5":    "f670b69b3d123fa53e3e1848a0b6bf6b",
	"SHA1":   "da39a3ee5e6b4b0d3255bfef95601890afd80709",
	"SHA224": "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f",
	"SHA384": "38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b",
	"SHA512": "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
	"SHA256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
}

func TestNewHashSum(t *testing.T) {
	hashAlgos := []string{"MD5", "SHA1", "SHA224", "SHA384", "SHA512", "SHA256"}
	file, err := os.OpenFile(fp, os.O_RDONLY, 0)
	if err != nil {
		log.Printf("Error reading file[%s]: %s", fp, err)
		t.Fail()
		return
	}
	for _, v := range hashAlgos {
		hashSummator := NewHashSummator(v)
		_, err = io.Copy(hashSummator, file)
		if err != nil {
			log.Printf("Error reading file[%s] with algo[%s]: %s", fp, v, err)
		}
		res := hex.EncodeToString(hashSummator.Sum(nil))
		if res != defuletValues[v] {
			log.Printf("New hash not corresponding to default value. %s(new hash) != %s(default value)", res, defuletValues[v])
			t.Fail()
			return
		}
	}
	log.Printf("LGTM, I'm thinking")
}
