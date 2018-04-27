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

package piazza

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func TestUuid(t *testing.T) {
	assert := assert.New(t)

	u := NewUuid()
	assert.NotNil(u)
	assert.True(u.Valid())

	s := fmt.Sprintf("%s", u.String())

	x := "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	assert.Len(x, 36)
	assert.Len(s, 36)

	assert.False(ValidUuid(x[1:34]))
}
