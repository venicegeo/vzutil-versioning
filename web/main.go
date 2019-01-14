// Copyright 2018, RadiantBlue Technologies, Inc.
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

package main

import (
	"log"

	"github.com/venicegeo/pz-gocommon/elasticsearch"
	"github.com/venicegeo/vzutil-versioning/web/app"
	a "github.com/venicegeo/vzutil-versioning/web/app/auth"
	s "github.com/venicegeo/vzutil-versioning/web/app/structs"
)

func main() {
	auth, err := a.NewAuthManager()
	if err != nil {
		log.Fatalln(err)
	}

	url, user, pass, err := s.GetVcapES()
	log.Printf("The elasticsearch url has been found to be [%s]\n", url)
	if err != nil {
		log.Fatalln(err)
	}
	index, err := elasticsearch.NewIndex2(url, user, pass, "versioning_tool", app.ESMapping)
	if err != nil {
		log.Fatalln(err.Error())
	} else {
		log.Println(index.GetVersion())
	}

	app := app.NewApplication(index, auth, "./single", "./compare", "templates/", false)
	app.StartInternals()
	log.Println(<-app.StartServer())
}
