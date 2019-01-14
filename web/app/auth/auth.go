// Copyright 2019, RadiantBlue Technologies, Inc.
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

package auth

import (
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"html/template"
	"os"
	"reflect"
	"strings"
)

type KeyForm struct {
	Key string `form:"key"`
}

type UserPassForm struct {
	User string `form:"user"`
	Pass string `form:"pass"`
}

const (
	key_html = template.HTML(`
<form method="post">
	Key:
	<input type="password" name="key">
	<input type="submit" name="button_submit" value="Submit">
</form>`)
)

type AuthManager struct {
	mode  string
	check checker
}

func NewAuthManager() (*AuthManager, error) {
	mode := os.Getenv("VZUTIL_AUTH_MODE")
	var check checker
	var err error = nil
	switch strings.ToUpper(mode) {
	case "ENVKEY":
		check, err = newEnvKeyChecker()
	case "OAUTH":
	case "LDAP":
	case "MYSQL":
	default:
		return nil, errors.New("Unrecognized auth mode")
	}
	if err != nil {
		return nil, err
	}
	return &AuthManager{mode, check}, nil
}

func (a *AuthManager) Check(i interface{}) (bool, error) {
	return a.check.check(i)
}
func (a *AuthManager) GetForm() interface{} {
	return a.check.getForm()
}
func (a *AuthManager) GetHTML() template.HTML {
	return a.check.getHTML()
}

type checker interface {
	check(interface{}) (bool, error)
	getForm() interface{}
	getHTML() template.HTML
}

type envkey_checker struct {
	key string
}

func newEnvKeyChecker() (*envkey_checker, error) {
	key := os.Getenv("VZUTIL_AUTH_KEY")
	l := len([]byte(key)) * 4
	if l != 256 && l != 512 {
		return nil, errors.New("Invalid key size")
	}
	return &envkey_checker{key}, nil
}

func (e *envkey_checker) check(form interface{}) (bool, error) {
	fmt.Println("Checking")
	fmt.Println(form)
	fmt.Println(reflect.TypeOf(form))
	f, ok := form.(*KeyForm)
	if !ok {
		return false, errors.New("Bad form")
	}
	var k string
	if len([]byte(e.key))*4 == 256 {
		h := sha256.Sum256([]byte(f.Key))
		k = fmt.Sprintf("%x", h[:])
	} else {
		h := sha512.Sum512([]byte(f.Key))
		k = fmt.Sprintf("%x", h[:])
	}
	return e.key == k, nil
}

func (e *envkey_checker) getForm() interface{} {
	return new(KeyForm)
}
func (e *envkey_checker) getHTML() template.HTML {
	return key_html
}
