/*
Copyright 2018, RadiantBlue Technologies, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package resolve

import (
	"testing"

	d "github.com/venicegeo/vzutil-versioning/common/dependency"
	i "github.com/venicegeo/vzutil-versioning/common/issue"
	l "github.com/venicegeo/vzutil-versioning/common/language"
)

func TestPomXml(t *testing.T) {
	addTest("pom_xml", `
<project>
	<dependencies>
		<dependency>
			<groupId>some.place</groupId>
			<artifactId>spring</artifactId>
			<version>1.4</version>
		</dependency>
		<dependency>
			<groupId>another.place</groupId>
			<artifactId>mock</artifactId>
			<version>1.Release</version>
		</dependency>
	</dependencies>
</project>
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("mock", "1.release", l.Java), d.NewDependency("spring", "1.4", l.Java)},
		issues: i.Issues{i.NewIssue("Failed to build [] with maven")},
		err:    nil,
	}, resolver.ResolvePomXml)

	addTest("pom_xml", `
<project>
	<parent>
		<groupId>spring.group</groupId>
		<artifactId>spring-parent</artifactId>
		<version>1.2.RELEASE</version>
	</parent>
	<dependencies>
		<dependency>
			<groupId>some.place</groupId>
			<artifactId>spring</artifactId>
			<version>1.4</version>
		</dependency>
		<dependency>
			<groupId>another.place</groupId>
			<artifactId>mock</artifactId>
			<scope>compile</scope>
		</dependency>
	</dependencies>
	<build>
		<plugins>
			<plugin>
				<groupId>maven.group</groupId>
				<artifactId>spring-maven</artifactId>
			</plugin>
		</plugins>
	</build>
</project>
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("mock", "", l.Java), d.NewDependency("spring", "1.4", l.Java), d.NewDependency("spring-maven", "", l.Java), d.NewDependency("spring-parent", "1.2.release", l.Java)},
		issues: i.Issues{i.NewIssue("Failed to build [] with maven"), i.NewMissingVersion("mock"), i.NewMissingVersion("spring-maven")},
		err:    nil,
	}, resolver.ResolvePomXml)

	run("pom_xml", t)

}
