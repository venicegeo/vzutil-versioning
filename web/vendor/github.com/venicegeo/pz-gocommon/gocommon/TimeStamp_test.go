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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

//--------------------------

func TestTimeStamp01(t *testing.T) {
	assert := assert.New(t)

	ts1 := NewTimeStamp()
	s1 := ts1.String()
	//log.Printf("*** %s", s1)
	ts2, err := ParseTimeStamp(s1)
	assert.NoError(err)
	s2 := ts2.String()
	//log.Printf("*** %s", s2)

	assert.EqualValues(s1, s2)

	byts, err := ts1.MarshalJSON()
	assert.NoError(err)
	//log.Printf("%s", string(byts))
	err = ts2.UnmarshalJSON(byts)
	assert.NoError(err)
	//log.Printf("*** %s", ts2.String()[:19])

	assert.EqualValues(ts1.String()[:19], ts2.String()[:19])
}

func TestTimeStamp02(t *testing.T) {
	assert := assert.New(t)

	var err error

	s := `"2017-01-12T23:35:24Z"`
	//log.Printf("A: %s", s)

	ts := &TimeStamp{}
	err = ts.UnmarshalJSON([]byte(s))
	assert.NoError(err)

	//log.Printf("B: %s", ts2.String())

	tt := time.Time(*ts)
	assert.False(tt.IsZero())

	assert.EqualValues(s, `"`+tt.Format(time.RFC3339)+`"`)
	assert.EqualValues(s, `"`+ts.String()+`"`)
}
