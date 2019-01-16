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

package app

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

type DIR uint8

const (
	UP DIR = iota
	DOWN
)

func (a *Application) repoConcept(c *gin.Context) {
	var temp struct {
		Sha string `form:"sha"`
	}
	c.Bind(&temp)

	type VNode struct {
		Id             string `json:"id"`
		Label          string `json:"label"`
		SecondaryLabel string `json:"labelSecondary"`
		Level          int    `json:"level"`
	}
	type VEdge struct {
		To     string `json:"to"`
		From   string `json:"from"`
		Arrows string `json:"arrows"`
	}
	nodes := []VNode{}
	edges := []VEdge{}
	/*roots*/ _, leaves, tree := generateTree(DELETETHIS(), 0)
	//for _, n := range tree {
	//	nodes = append(nodes, VNode{n.Sha, n.Sha})
	//}
	sub := subtree(tree, leaves, 1, 3)
	fmt.Println(leaves)
	fmt.Println(sub)
	fmt.Println(len(sub))
	reset_tree(tree, 0)
	for _, root := range leaves {
		fmt.Println(root)
		traverse(sub, root, 1, DOWN, func(n *Node) {
			for _, p := range n.Parents {
				fmt.Println("Connecting", p, "to", n.Sha)
				edges = append(edges, VEdge{n.Sha, p, "to"})
			}
		})
	}
	reset_tree(tree, -1)
	max := -1
	for _, l := range leaves {
		temp := calc_weights(sub, l, 0)
		if temp > max {
			max = temp
		}
	}
	for _, v := range sub {
		v.flag = max - v.flag
	}
	for _, n := range sub {
		nodes = append(nodes, VNode{n.Sha, n.Sha, n.Name, n.flag})
	}
	//	for _, root := range roots {
	//		traverse(tree, root, true, func(n *Node) {
	//			for _, c := range n.Children {
	//				edges = append(edges, VEdge{c, n.Sha, "to"})
	//			}
	//		})
	//	}
	c.HTML(200, "repo_overview_concept.html", gin.H{"nodes": nodes, "edges": edges})
}

func calc_weights(tree map[string]*Node, root string, setTo int) int {
	node := tree[root]
	if node == nil {
		return setTo
	}
	max := setTo
	if node.flag >= setTo {
		return max
	}
	node.flag = setTo
	temp := max
	for _, p := range node.Parents {
		temp = calc_weights(tree, p, setTo+1)
		if temp > max {
			max = temp
		}
	}
	return max
}

func reset_tree(tree map[string]*Node, setFlag int) {
	for _, n := range tree {
		n.flag = setFlag
	}
}

func traverse(tree map[string]*Node, root string, setFlag int, direction DIR, todo func(*Node)) {
	fmt.Println("Looking at", root)
	if tree[root] == nil || tree[root].flag == setFlag {
		fmt.Println(tree[root])
		fmt.Println("returning")
		return
	}
	tree[root].flag = setFlag
	todo(tree[root])
	next := tree[root].Parents
	if direction == UP {
		next = tree[root].Children
	}
	fmt.Println("Moving to", next)
	for _, c := range next {
		traverse(tree, c, setFlag, direction, todo)
	}
}

func subtree(tree map[string]*Node, roots []string, setFlag int, count int) map[string]*Node {
	//fmt.Println("roots", roots)
	res := make(map[string]*Node)
	for _, r := range roots {
		subtree_s(tree, r, setFlag, count, res)
	}
	return res
}
func subtree_s(tree map[string]*Node, root string, setFlag int, count int, res map[string]*Node) {
	if count == 0 || tree[root].flag == setFlag {
		//fmt.Println("Not doing", root)
		return
	}
	tree[root].flag = setFlag
	res[root] = tree[root]
	//fmt.Println(root, tree[root].Parents)
	for _, c := range tree[root].Parents {
		subtree_s(tree, c, setFlag, count-1, res)
	}
}

type Node struct {
	Sha      string
	Parents  []string
	Children []string
	Name     string
	flag     int
}

func generateTree(data string, flag int) ([]string, []string, map[string]*Node) {
	nodes := map[string]*Node{}
	roots := []string{}
	leafs := []string{}
	lines := strings.Split(strings.TrimSpace(data), "\n")
	parts := make([][]string, len(lines))
	for i := 0; i < len(lines); i++ {
		parts[i] = strings.Split(lines[i], "|")
		nodes[parts[i][0]] = &Node{parts[i][0], []string{}, []string{}, parts[i][2], flag}
	}
	for i := len(parts) - 1; i >= 0; i-- {
		line := parts[i]
		node := nodes[line[0]]
		parents := strings.Split(line[1], " ")
		for _, p := range parents {
			if p == "" {
				continue
			}
			node.Parents = append(node.Parents, p)
			nodes[p].Children = append(nodes[p].Children, node.Sha)
		}
	}
	for s, v := range nodes {
		if len(v.Parents) == 0 {
			roots = append(roots, s)
		}
	}
	for s, v := range nodes {
		if len(v.Children) == 0 {
			leafs = append(leafs, s)
		}
	}
	return roots, leafs, nodes
}

var treeTestData = `
ghi|abc|
def|abc|
abc||`
