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

	// printf("-DEBUG: bsumHashFile(): hash: [ ");
	// for (int i=0; i<64; i++) {
	// 	printf("%d ", hash[i]);
	// }
	// printf(" ]\n");

	return 0;
}
*/
import "C"

import (
	"unsafe"

	"github.com/sirupsen/logrus"
)

// The bee2 configuration parameters
const (
	// The depth (or strench) of algorithm. The valid values are 32..512 with
	// step 32.
	HID C.ulong = 256
	// The default value for the algorithm is 64. But it may not use all the
	// memory. Real usage depends on HID value and calculates as HID/8.
	HashSize = 64
)

// Bee2HashFile returns hash of fname file
func Bee2HashFile(fname string, log *logrus.Logger) string {
	fnameC := C.CString(fname)
	defer C.free(unsafe.Pointer(fnameC))

	var buf [HashSize]byte
	hashC := C.CBytes(buf[:])
	defer C.free(hashC)

	arr64 := (*C.uchar)(hashC)
	C.bsumHashFile(arr64, HID, fnameC)

	bytesFromC := C.GoBytes(unsafe.Pointer(arr64), HashSize)
	hash := string(bytesFromC)
	log.WithFields(logrus.Fields{
		"name":          fname,
		"hash (string)": hash,
		"hash (bytes)":  []byte(hash),
	}).Debug("file")

	return hash
}
