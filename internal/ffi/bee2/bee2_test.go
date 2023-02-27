package bee2

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBee2(t *testing.T) {
	assert.Equal(t,
		&bee2{
			hid:      HID,
			hashSize: HASHSIZE,
			data:     []byte{},
			hash:     make([]byte, HASHSIZE),
			state:    make([]byte, BLOCKSIZE),
		},
		New(),
	)
	assert.Equal(t,
		&bee2{
			hid:      256,
			hashSize: 32,
			data:     []byte{},
			hash:     make([]byte, HASHSIZE),
			state:    make([]byte, BLOCKSIZE),
		},
		New(WithHid(256), WithHashSize(32)),
	)
}

func Test_bee2_Write(t *testing.T) {
	testString := "test bytes"
	b2 := New()
	b2.Write([]byte(testString))

	s := string(b2.(*bee2).data)
	assert.Equal(t, s, testString)
}
