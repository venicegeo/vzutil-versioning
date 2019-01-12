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
	"strings"

	"github.com/gin-gonic/gin"
)

func (a *Application) repoConcept(c *gin.Context) {
	var temp struct {
		Sha string `form:"sha"`
	}
	c.Bind(&temp)
	fmt.Println(temp)
	type VNode struct {
		Id    string `json:"id"`
		Label string `json:"label"`
	}
	type VEdge struct {
		To     string `json:"to"`
		From   string `json:"from"`
		Arrows string `json:"arrows"`
	}
	nodes := []VNode{}
	edges := []VEdge{}
	roots, all := generateTree(treeTestData, false)
	for _, n := range all {
		nodes = append(nodes, VNode{n.Sha, n.Sha})
	}
	for _, root := range roots {
		traverse(root, true, func(n *Node) {
			for _, c := range n.Children {
				edges = append(edges, VEdge{c.Sha, n.Sha, "to"})
			}
		})
	}
	c.HTML(200, "repo_overview_concept.html", gin.H{"nodes": nodes, "edges": edges})
}

func traverse(root *Node, setFlag bool, todo func(*Node)) {
	if root.flag == setFlag {
		return
	}
	root.flag = setFlag
	todo(root)
	for _, c := range root.Children {
		traverse(c, setFlag, todo)
	}
}

type Node struct {
	Sha      string
	Parents  []*Node
	Children []*Node
	Name     string
	flag     bool
}

func generateTree(data string, flag bool) ([]*Node, map[string]*Node) {
	nodes := map[string]*Node{}
	roots := []*Node{}
	lines := strings.Split(strings.TrimSpace(data), "\n")
	parts := make([][]string, len(lines))
	for i := 0; i < len(lines); i++ {
		parts[i] = strings.Split(lines[i], "|")
		nodes[parts[i][0]] = &Node{parts[i][0], []*Node{}, []*Node{}, parts[i][2], flag}
	}
	for i := len(parts) - 1; i >= 0; i-- {
		line := parts[i]
		node := nodes[line[0]]
		parents := strings.Split(line[1], " ")
		for _, p := range parents {
			if p == "" {
				continue
			}
			node.Parents = append(node.Parents, nodes[p])
			nodes[p].Children = append(nodes[p].Children, node)
		}
	}
	for _, v := range nodes {
		fmt.Println(*v)
		if len(v.Parents) == 0 {
			roots = append(roots, v)
		}
	}
	return roots, nodes
}

var treeTestData = `
ghi|abc|
def|abc|
abc||`
