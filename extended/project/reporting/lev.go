/*
Copyright 2017, RadiantBlue Technologies, Inc.

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
package report

import (
	"strings"
)

func similaritySort(a, b []string) ([]string, []string) {
	if len(a) != len(b) {
		panic("Slices must be of same length")
	}
	remove := func(slice []string, s int) []string {
		return append(slice[:s], slice[s+1:]...)
	}
	resultA := make([]string, len(a))
	resultB := make([]string, len(a))
	recCount := 0
	sort := func() {
		size := len(a)
		similarityMatrix := make([][]float64, size)
		for i, av := range a {
			similarityMatrix[i] = make([]float64, size)
			for j, bv := range b {
				similarityMatrix[i][j] = similarity(av, bv)
			}
		}

		rm, cm := 0, 0
		for r := 0; r < size; r++ {
			for c := 0; c < size; c++ {
				if similarityMatrix[r][c] > similarityMatrix[rm][cm] {
					rm = r
					cm = c
				}
			}
		}
		resultA[recCount] = a[rm]
		resultB[recCount] = b[cm]
		a = remove(a, rm)
		b = remove(b, cm)
		recCount++
	}
	for len(a) > 0 {
		sort()
	}

	return resultA, resultB
}

func similarity(s, t string) float64 {
	if strings.TrimSpace(s) == "" && strings.TrimSpace(t) == "" {
		return 0
	}
	longer, shorter := s, t
	if len(longer) < len(shorter) {
		longer, shorter = t, s
	}
	longerLength := len(longer)
	if longerLength == 0 {
		return 1.0
	}
	return float64(longerLength-levenshteinDistance(longer, shorter)) / float64(longerLength)
}

func levenshteinDistance(s, t string) int {
	s = strings.ToLower(s)
	t = strings.ToLower(t)

	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	costs := make([]int, len(t)+1)
	for i := 0; i <= len(s); i++ {
		lastValue := i
		for j := 0; j <= len(t); j++ {
			if i == 0 {
				costs[j] = j
			} else {
				if j > 0 {
					newValue := costs[j-1]
					if s[i-1] != t[j-1] {
						newValue = min(min(newValue, lastValue), costs[j]) + 1
					}
					costs[j-1] = lastValue
					lastValue = newValue
				}
			}
		}
		if i > 0 {
			costs[len(t)] = lastValue
		}
	}
	return costs[len(t)]
}
