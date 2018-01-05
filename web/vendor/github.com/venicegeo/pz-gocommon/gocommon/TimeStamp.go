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
	"log"
	"time"
)

// TimeStamp is used for the timeStamp field of a syslog.Message,
// but it is here in pz-gocommon because it may be generally useful.
//
// A Piazza timestamp must always be in RFC3339 format and in UTC. That is,
// we use yyyy-MM-dd'T'HH:mm:ss.SSSZ, with three caveats:
// (1) the ".SSS" (fractional seconds) part is optional
// (2) if the fractional part is present, there may actually be 1-7 digits used
// (3) however, if the fractional part is present when reading in or writing out, then
// the system may silently truncate or round to only three digits (milliseconds).
//
// There are three basic operations: Now, Parse, and String.
// In these functions, note how we always are careful to round the time to milliseconds
// and then convert it to UTC before proceeding with any usages of
// Time objects or time strings.
//
// Note that when Go marshals a Time object into JSON, it uses the "RFC3339Nano"
// format which may have several fractional digits. Note also that when storing a
// time object into Elasticsearch, the entire fractional part may be truncated to
// zero fractional digits (second accuracy) or three fractional digits (millisecond
// accuracy).
type TimeStamp time.Time

func NewTimeStamp() TimeStamp {
	t := time.Now().Round(time.Millisecond).UTC()
	return TimeStamp(t)
}

func ParseTimeStamp(s string) (TimeStamp, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return TimeStamp{}, err
	}
	t = t.Round(time.Millisecond).UTC()
	return TimeStamp(t), nil
}

func (ts TimeStamp) String() string {
	t := time.Time(ts)
	t = t.Round(time.Millisecond).UTC() // just in case
	return t.Format(time.RFC3339)
}

func (ts TimeStamp) MarshalJSON() ([]byte, error) {
	t := time.Time(ts)
	return t.MarshalJSON()
}

func (ts *TimeStamp) UnmarshalJSON(data []byte) error {
	t := (*time.Time)(ts)
	err := t.UnmarshalJSON(data)
	return err
}

func (ts *TimeStamp) Validate() error {
	t := time.Time(*ts)
	if t.IsZero() {
		return fmt.Errorf("TimeStamp is zero")
	}
	if t.Location() != time.UTC {
		// TODO: turn on enforcement of this
		log.Printf("WARNING: TimeStamp is not in UTC [%s]", t.String())
		////return fmt.Errorf("TimeStamp is not in UTC")
	}
	return nil
}
