// Copyright 2016, RadiantBlue Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This code is derived from https://github.com/pborman/uuid which is
//   Copyright 2011 Google Inc.  All rights reserved.
//   Use of this source code is governed by a BSD-style
//   license that can be found in the LICENSE file.

package piazza

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

type Uuid []byte

var rander = rand.Reader // random function

// encodeHex makes a string (in bytes) from the aray of uuid bytes.
func encodeHex(dst []byte, uuid Uuid) {
	// hex.Encode takes an input byte array and returns the bytes of the string
	// of the hex encoding of each input byte
	// example: [63,127] ==> ['3', 'f', '7', 'f']
	hex.Encode(dst[:], uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
}

// randomBits completely fills slice b with random data.
func randomBits(b []byte) {
	if _, err := io.ReadFull(rander, b); err != nil {
		panic(err.Error()) // rand should never fail
	}
}

// String returns the string form of uuid: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
func (uuid Uuid) String() string {
	if len(uuid) != 16 {
		panic(fmt.Errorf("invalid uuid length"))
	}
	var buf [36]byte
	encodeHex(buf[:], uuid)
	return string(buf[:])
}

// NewRandom returns a random (variant 4) UUID.
func NewUuid() Uuid {
	uuid := make([]byte, 16)
	randomBits(uuid)
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	return uuid
}

func ValidUuid(uuid string) bool {
	if len(uuid) != 36 || uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		return false
	}
	return true
}

func (uuid Uuid) Valid() bool {
	return ValidUuid(uuid.String())
}
