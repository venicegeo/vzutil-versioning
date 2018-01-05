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

package kafka

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type KafkaTester struct {
	suite.Suite
}

func (suite *KafkaTester) SetupSuite() {}

func (suite *KafkaTester) TearDownSuite() {}

func TestRunSuite(t *testing.T) {
	s1 := new(KafkaTester)
	suite.Run(t, s1)
}

//=================================================================

func (suite *KafkaTester) Test01() {
	Stub()
}
