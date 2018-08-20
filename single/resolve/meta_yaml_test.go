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

func TestMetaYaml(t *testing.T) {
	addTest("meta_yaml", `
package:
  name: some-package
  version: 1.0a

source:
  path: ./some-package

requirements:
  build:
    - setuptools 39.2.0
    - numpy 1.14.0 py27_blas_openblas_200
    - python 2.7.13
  run:
    - numpy 1.14.0 py27_blas_openblas_200
    - python 2.7.13
    - gdal 2.1.3

build:
  preserve_egg_dir: True
  string: py{{py}}_0
  #binary_relocation: False

about:
  home: https://github.com/venicegeo/bfalg-ndwi
  summary: "A library and a CLI for running shoreline detection "
  license: Apache 2.0
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("gdal", "2.1.3", l.Conda), d.NewDependency("numpy", "1.14.0=py27_blas_openblas_200", l.Conda), d.NewDependency("python", "2.7.13", l.Conda), d.NewDependency("setuptools", "39.2.0", l.Conda)},
		issues: i.Issues{},
		err:    nil,
	}, resolver.ResolveMetaYaml)

	addTest("meta_yaml", `
{% set version = "1.14.3" %}
{% set variant = "openblas" %}

package:
  name: numpy
  version: {{ version }}

build:
  number: 200
  skip: true  # [win32 or (win and py27)]
  features:
    - blas_{{ variant }}

requirements:
  host:
    - python
    - pip
    - cython
    - blas 1.1 {{ variant }}
    - openblas
  run:
    - python
    - blas 1.1 {{ variant }}
    - openblas
`, ResolveResult{
		deps:   d.Dependencies{d.NewDependency("blas", "1.1=openblas", l.Conda), d.NewDependency("cython", "", l.Conda), d.NewDependency("openblas", "", l.Conda), d.NewDependency("pip", "", l.Conda), d.NewDependency("python", "", l.Conda)},
		issues: i.Issues{i.NewMissingVersion("cython"), i.NewMissingVersion("openblas"), i.NewMissingVersion("openblas"), i.NewMissingVersion("pip"), i.NewMissingVersion("python"), i.NewMissingVersion("python")},
		err:    nil,
	}, resolver.ResolveMetaYaml)

	run("meta_yaml", t)

}
