/*
Copyright 2019, RadiantBlue Technologies, Inc.

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
package history

import (
	"sort"
)

type DIR uint8

const (
	UP DIR = iota
	DOWN
)

type HistoryTree map[string]*HistoryNode

type HistoryNode struct {
	Sha             string   `json:"sha"`
	Parents         []string `json:"parents"`
	Children        []string `json:"children"`
	Names           []string `json:"names"`
	Weight          int      `json:"weight"`
	IsHEAD          bool     `json:"isHEAD"`
	IsStartOfBranch bool     `json:"isStartOfBranch"`
}

func (n *HistoryNode) duplicate() *HistoryNode {
	res := new(HistoryNode)
	res.Sha = n.Sha
	res.Weight = n.Weight
	res.IsHEAD = n.IsHEAD
	res.IsStartOfBranch = n.IsStartOfBranch

	res.Parents = make([]string, len(n.Parents))
	copy(res.Parents, n.Parents)
	res.Children = make([]string, len(n.Children))
	copy(res.Children, n.Children)
	res.Names = make([]string, len(n.Names))
	copy(res.Names, n.Names)

	return res
}

type byLength []string

func (s byLength) Len() int {
	return len(s)
}
func (s byLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byLength) Less(i, j int) bool {
	if len(s[i]) != len(s[j]) {
		return len(s[i]) > len(s[j])
	}
	return s[i] > s[j]
}

func SortByDecreasingLength(s []string) {
	sort.Sort(byLength(s))
}

//Nodes with no parents, usually only 1
func (t *HistoryTree) GetRoots() []string {
	roots := make([]string, 0, 1)
	for _, node := range *t {
		hasParent := false
		for _, ps := range node.Parents {
			if _, ok := (*t)[ps]; ok {
				hasParent = true
				break
			}
		}
		if !hasParent {
			roots = append(roots, node.Sha)
		}
	}
	return roots
}

//Nodes with no children, usually number of branches
func (t *HistoryTree) GetLeafs() []string {
	leafs := []string{}
	for _, node := range *t {
		hasChild := false
		for _, cs := range node.Children {
			if _, ok := (*t)[cs]; ok {
				hasChild = true
				break
			}
		}
		if !hasChild {
			leafs = append(leafs, node.Sha)
		}
	}
	return leafs
}

func (t *HistoryTree) ResetAllWeights(weight int) {
	for _, n := range *t {
		n.Weight = weight
	}
}

func (t *HistoryTree) TraverseFrom(root string, dir DIR, weight int, todo func(*HistoryNode, int) (bool, int)) {
	if (*t)[root] == nil {
		return
	}
	cont, nextWeight := todo((*t)[root], weight)
	if !cont {
		return
	}
	for _, n := range (*t)[root].getNext(dir) {
		t.TraverseFrom(n, dir, nextWeight, todo)
	}
}

//RESETS WEIGHTS TO 0
func (t *HistoryTree) GenerateSubtree(roots []string, dir DIR, depth int) HistoryTree {
	res := HistoryTree{}
	t.ResetAllWeights(0)
	for _, root := range roots {
		t.generateSubtree(res, root, dir, depth)
	}
	return res
}
func (t *HistoryTree) generateSubtree(res HistoryTree, root string, dir DIR, depth int) {
	node := (*t)[root]
	if node == nil || node.Weight == 1 || depth == 0 {
		return
	}
	res[root] = node.duplicate()
	node.Weight = 1
	for _, n := range node.getNext(dir) {
		t.generateSubtree(res, n, dir, depth-1)
	}
}

func (t *HistoryNode) getNext(dir DIR) []string {
	switch dir {
	case UP:
		return t.Parents
	case DOWN:
		return t.Children
	default:
		return nil
	}
}

func (t *HistoryTree) CalculateHeights(root string, dir DIR, weight int) int {
	node := (*t)[root]
	if node == nil {
		return weight
	}
	max := weight
	if node.Weight >= weight {
		return max
	}
	node.Weight = weight
	temp := max
	for _, n := range node.getNext(dir) {
		temp = t.CalculateHeights(n, dir, weight+1)
		if temp > max {
			max = temp
		}
	}
	return max
}

func (t *HistoryTree) MaxWeight() int {
	if len(*t) == 0 {
		return 0
	}
	var max int
	for _, n := range *t {
		max = n.Weight
		break
	}
	for _, n := range *t {
		if n.Weight > max {
			max = n.Weight
		}
	}
	return max
}

func (t *HistoryTree) ReverseWeights(reverseFrom int) {
	for _, n := range *t {
		n.Weight = reverseFrom - n.Weight
	}
}

func (t *HistoryTree) FindMissingNames(fullTree HistoryTree) []string {
	allNames := map[string]string{}
	for _, n := range fullTree {
		for _, name := range n.Names {
			allNames[name] = n.Sha
		}
	}
	for _, n := range *t {
		for _, name := range n.Names {
			delete(allNames, name)
		}
	}
	temp := map[string]struct{}{}
	for _, v := range allNames {
		temp[v] = struct{}{}
	}
	res := make([]string, 0, len(temp))
	for k, _ := range temp {
		res = append(res, k)
	}
	return res
}
