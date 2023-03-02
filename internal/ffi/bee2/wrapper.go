//go:build bee2

package bee2

/*
  The Go wrapper for the bee2 library.
  Original library repo: https://github.com/agievich/bee2

  Before compile this module the bee2 library build is prerequested.
  See Readme.md file in the local directory for the library build instractions.
*/

/*
#cgo CFLAGS: -I../../../bee2/include
#cgo LDFLAGS: -L../../../bee2/build/src -l:libbee2_static.a

#include <stdlib.h>
#include <stdio.h>

#include "bee2/crypto/bash.h"
#include "bee2/crypto/belt.h"

typedef unsigned char octet;

// The code has benn taken from: github.com/agievich/bee2/cmd/bsum/bsum.c:86
// and contains some minor changes
int bsumHashFile(octet hash[], size_t hid, const char* filename)
{
	FILE* fp;
	octet state[4096];
	octet buf[4096];
	size_t count;

	fp = fopen(filename, "rb");
	if (!fp)
	{
		printf("%s: FAILED [open]\n", filename);
		return -1;
	}

	// ASSERT(beltHash_keep() <= sizeof(state));
	// ASSERT(bashHash_keep() <= sizeof(state));
	hid ? bashHashStart(state, hid / 2) : beltHashStart(state);

	while (1)
	{
		count = fread(buf, 1, sizeof(buf), fp);
		if (count == 0)
		{
			if (ferror(fp))
			{
				fclose(fp);
				printf("%s: FAILED [read]\n", filename);
				return -1;
			}
			break;
		}
		hid ? bashHashStepH(buf, count, state) :
			beltHashStepH(buf, count, state);
	}
	fclose(fp);
	hid ? bashHashStepG(hash, hid / 8, state) : beltHashStepG(hash, state);
	return 0;
}
*/
import "C"

import (
	"encoding/hex"
	"fmt"
	"unsafe"
)

// The bee2 configuration parameters
const (
	// The depth (or strench) of algorithm. The valid values are 32..512 with
	// step 32.
	HID int = 256
	// The default value for the algorithm is 64. But it may not use all the
	// memory. Real usage depends on HID value and calculates as HID/8.
	HASHSIZE = 32
	// Bytes, len of block for data processing
	BLOCKSIZE = 4096
)

// Bee2HashFile returns hash of fname file
func Bee2HashFile(fname string) (string, error) {
	fnameC := C.CString(fname)
	defer C.free(unsafe.Pointer(fnameC))

	var buf [HASHSIZE]byte
	hashC := C.CBytes(buf[:])
	defer C.free(hashC)

	arr := (*C.uchar)(hashC)
	errCode := C.bsumHashFile(arr, C.ulong(HID), fnameC)
	if int(errCode) != 0 {
		return "", fmt.Errorf("file not found")
	}
	bytesFromC := C.GoBytes(unsafe.Pointer(arr), HASHSIZE)

	return hex.EncodeToString(bytesFromC), nil
}

func (a *bee2) bashHashStart() {
	stateC := C.CBytes(a.state)
	defer C.free(stateC)

	C.bashHashStart(stateC, C.ulong(a.hid/2))
	copy(a.state, C.GoBytes(stateC, BLOCKSIZE))
}

func (a *bee2) bashHashStepH() {
	stateC := C.CBytes(a.state)
	defer C.free(stateC)

	bufC := C.CBytes(a.data)
	defer C.free(bufC)

	C.bashHashStepH(bufC, C.ulong(a.n), stateC)
	copy(a.state, C.GoBytes(stateC, BLOCKSIZE))
}

func (a *bee2) bashHashStepG() {
	stateC := C.CBytes(a.state)
	defer C.free(stateC)

	hashC := C.CBytes(a.hash)
	defer C.free(hashC)

	C.bashHashStepG((*C.uchar)(hashC), C.ulong(a.hid/8), stateC)
	copy(a.hash, C.GoBytes(hashC, C.int(a.hashSize)))
}
