// Copyright 2017, RadiantBlue Technologies, Inc.
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

//action auditString
func RequestAuthZAccess(idamURL string, actor string) (authorized bool, err error) {
	//                         \/ Subject to change \/
	req := fmt.Sprintf(`{
							"username": "%s",
							"action": {
								"requestMethod": "POST",
								"uri": "job" 
							}
						}`, actor)
	code, body, _, err := HTTP(POST, idamURL+"/authz", NewHeaderBuilder().AddJsonContentType().GetHeader(), bytes.NewReader([]byte(req)))
	if err != nil {
		return false, err
	}
	if code != 200 {
		return false, errors.New("AuthZ response code not 200")
	}
	var respon map[string]interface{}
	if err = json.Unmarshal(body, &respon); err != nil {
		return false, err
	}
	if iAuth, ok := respon["isAuthSuccess"]; !ok {
		return false, errors.New("AuthZ response doesn't contain isAuthSuccess field")
	} else if authorized, ok = iAuth.(bool); !ok {
		return false, errors.New("AuthZ isAuthSuccess is not type bool")
	}
	return authorized, nil
}
