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
	"bytes"
	"encoding/json"

	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	t "github.com/venicegeo/vzutil-versioning/web/es/types"

	nt "github.com/venicegeo/pz-gocommon/gocommon"
	"github.com/venicegeo/vzutil-versioning/web/es"
	u "github.com/venicegeo/vzutil-versioning/web/util"
)

type PipelineResponse struct {
	DisplayName string          `json:"displayName"`
	Builds      []PipelineBuild `json:"builds"`
}
type PipelineBuild struct {
	Number uint `json:"number"`
}

var findTimestamp = regexp.MustCompile(`.*"timestamp"\s*:\s*([0-9]+),`)

var sha_match = regexp.MustCompile(`^.*git checkout .*([a-f0-9]{40}).*$`)
var target_match = regexp.MustCompile(`^.*cf target (.*)$`)
var push_match = regexp.MustCompile(`^.*cf push (.*)$`)
var stage_match = regexp.MustCompile(`^\[Pipeline\] { \((.+)\)$`)

type JenkinsManager struct {
	app        *Application
	h          u.HTTP
	jenkinsUrl string
	authHeader [][2]string
}

func NewJenkinsManager(app *Application, h u.HTTP, jenkinsUrl string, authHeader [][2]string) *JenkinsManager {
	return &JenkinsManager{app, h, jenkinsUrl, authHeader}
}
func (m *JenkinsManager) Add(project, repository, url string) (string, error) {
	parts := strings.SplitN(url, "://", 2)
	if len(parts) == 2 {
		url = parts[1]
	}
	if !strings.HasPrefix(url, m.jenkinsUrl) {
		return "", u.Error("This is not the url expected")
	}
	url = strings.TrimPrefix(url, m.jenkinsUrl)
	jobParts := u.SplitAtAnyTrim(strings.ToLower(url), "job", "/")
	//TODO remove pipeline_info and replace with field name
	boool := es.NewBoolQ()
	for _, part := range jobParts {
		boool.Add(es.NewTerm(t.JenkinsPipeline_QField_PipelineInfo, part))
	}
	boool.Add(map[string]interface{}{
		"script": map[string]interface{}{
			"script": map[string]string{
				"inline": u.Format("doc['%s'].values.length == %d", t.JenkinsPipeline_QField_PipelineInfo, len(jobParts)),
				"lang":   "painless",
			},
		},
	})
	checkExistQ := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": es.NewBool().SetMust(boool),
		},
		"size": 0,
	}
	check, err := m.app.index.SearchByJSON(JenkinsPipeline_QType, checkExistQ)
	if err != nil {
		return "", err
	}
	if check.TotalHits() > 0 {
		return "", nil
	}
	id := nt.NewUuid().String()
	resp, err := m.app.index.PostDataWait(JenkinsPipeline_QType, id, &t.JenkinsPipeline{id, project, repository, jobParts})
	if err != nil {
		return "", err
	}
	if !resp.Created {
		return "", u.Error("The pipeline entry was not created for an unknown reason")
	}
	return id, nil
}

func (m *JenkinsManager) RunAutomatedScans(pause time.Duration, stop chan struct{}) {
	for {
		select {
		case <-stop:
			break
		default:
			if err := m.RunScan(); err != nil {
				log.Println("Error running jenkins scan:", err.Error())
			}
			time.Sleep(pause)
		}
	}
}

func (m *JenkinsManager) getAllEntries() ([]*t.JenkinsPipeline, error) {
	resp, err := es.GetAll(m.app.index, JenkinsPipeline_QType, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	res := make([]*t.JenkinsPipeline, len(resp.Hits), len(resp.Hits))
	for i, hit := range resp.Hits {
		entry := new(t.JenkinsPipeline)
		if err := json.Unmarshal([]byte(*hit.Source), entry); err != nil {
			return nil, err
		}
		res[i] = entry
	}
	return res, nil
}

func (m *JenkinsManager) getAllEntriesProj(projId string) ([]*t.JenkinsPipeline, error) {
	resp, err := es.GetAll(m.app.index, JenkinsPipeline_QType, es.NewTerm(t.JenkinsPipeline_QField_ProjectId, projId))
	if err != nil {
		return nil, err
	}
	res := make([]*t.JenkinsPipeline, len(resp.Hits), len(resp.Hits))
	for i, hit := range resp.Hits {
		entry := new(t.JenkinsPipeline)
		if err := json.Unmarshal([]byte(*hit.Source), entry); err != nil {
			return nil, err
		}
		res[i] = entry
	}
	return res, nil
}

func (m *JenkinsManager) RunScan() error {
	entries, err := m.getAllEntries()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		//log.Println("Looking at info:", entry.PipelineInfo)
		generalApi := u.Format("https://%s/job/%s/api/json?pretty=true", m.jenkinsUrl, strings.Join(entry.PipelineInfo, "/job/"))
		code, dat, _, err := m.h.HTTP(nt.GET, generalApi, m.authHeader, nil)
		if err != nil {
			return err
		} else if code != 200 {
			return u.Error("Code for general json api [%s] is [%d]", generalApi, code)
		}
		resp := new(PipelineResponse)
		if err = json.Unmarshal(dat, resp); err != nil {
			return err
		}
		//log.Println(resp)
		currentNewest, err := m.app.index.SearchByJSON(JenkinsBuildTargets_QType, map[string]interface{}{
			"size": 1,
			"sort": map[string]interface{}{
				"timestamp": "desc",
			},
		})
		if err != nil {
			return err
		}
		timeToBeat := "0"
		if len(currentNewest.Hits.Hits) == 1 {
			target := new(t.JenkinsBuildTargets)
			if err := json.Unmarshal([]byte(*currentNewest.Hits.Hits[0].Source), target); err != nil {
				return err
			}
			timeToBeat = target.Timestamp
		}
		//log.Println("Time to beat:", timeToBeat)
		allShasAdded := map[string]struct{}{}
		for _, build := range resp.Builds {
			//log.Println()
			//log.Println("Looking at build", build.Number)
			code, dat, _, err = m.h.HTTP(nt.GET, u.Format("https://%s/job/%s/%d/api/json?pretty=true", m.jenkinsUrl, strings.Join(entry.PipelineInfo, "/job/"), build.Number), m.authHeader, nil)
			if err != nil {
				return err
			} else if code != 200 {
				return u.Error("Code for api with info [%s] is [%d]", strings.Join(entry.PipelineInfo, " "), code)
			}
			var timestamp struct {
				Timestamp json.Number `json:"timestamp"`
			}

			if err = nt.UnmarshalNumber(bytes.NewReader(dat), &timestamp); err != nil {
				return err
			}
			tim := timestamp.Timestamp.String()
			//log.Println("Found timestamp to be", tim)
			if tim <= timeToBeat {
				//	log.Println("Reached outdated builds")
				break
			}
			code, dat, _, err = m.h.HTTP(nt.GET, u.Format("https://%s/job/%s/%d/consoleText", m.jenkinsUrl, strings.Join(entry.PipelineInfo, "/job/"), build.Number), m.authHeader, nil)
			if err != nil {
				return err
			} else if code != 200 {
				return u.Error("Console text for build [%s] is [%d]", strings.Join(entry.PipelineInfo, " "), code)
			}

			lines := u.SplitAtAnyTrim(string(dat), "\n", "\r")
			//			stages := m.runStageFA(lines)
			//			log.Println(stages)
			//			targets, sha := m.runDeployFA(lines, entry.Repository, stages)

			//HERE
			stageFA := u.NewFA()
			stages := []t.JenkinsBuildStage{}
			stageFA.Add("start", func(l string) bool {
				return l == `[Pipeline] stage`
			}, "next")
			stageFA.Add("next", func(l string) bool {
				stages = append(stages, t.JenkinsBuildStage{stage_match.FindStringSubmatch(l)[1], true})
				//fmt.Println("stages", stages)
				return true
			}, "end")
			stageFA.Add("end", func(l string) bool {
				return l == `[Pipeline] // stage`
			}, "start")
			stageFA.Every(func(l string) bool {
				if l == `Finished: FAILURE` {
					stages[len(stages)-1].Success = false
					return true
				}
				return false
			}, "")
			stageFA.Start("start")

			var sha string
			targets := []t.CFTarget{}
			deployFA := u.NewFA()
			repo, err := m.app.rtrvr.GetRepositoryById(entry.RepositoryId)
			if err != nil {
				return err
			}
			repo_match, err := regexp.Compile(u.Format(`^Cloning repository .+github.com\/%s.*$`, strings.Replace(repo.Fullname, "/", `\/`, -1)))
			if err != nil {
				panic(err)
			}
			//fmt.Println("CHECKOUT", repo_match.String())
			deployFA.Add("checkout", func(l string) bool {
				return repo_match.MatchString(l)
			}, "sha")
			deployFA.Add("sha", func(l string) bool {
				if sha_match.MatchString(l) {
					sha = sha_match.FindStringSubmatch(l)[1]
					return true
				}
				return false
			}, "target")
			deployFA.Add("target", func(l string) bool {
				if target_match.MatchString(l) {
					parts := u.SplitAtAnyTrim(target_match.FindStringSubmatch(l)[1], " ")
					target := t.CFTarget{}
					for i := 0; i < len(parts); i++ {
						if parts[i] == "-o" {
							target.Org = parts[i+1]
							i++
						} else if parts[i] == "-s" {
							target.Space = parts[i+1]
							i++
						}
					}
					targets = append(targets, target)
					//fmt.Println("targets", targets)
					return true
				}
				return false
			}, "target")
			deployFA.Add("target", func(l string) bool {
				if push_match.MatchString(l) {
					targets[len(targets)-1].Pushed = true
					targets[len(targets)-1].Stage = &stages[len(stages)-1]
					return true
				}
				return false
			}, "target")
			deployFA.Start("checkout")

			//			deployFA.RunAgainst(lines)
			//			stageFA.RunAgainst(lines)
			for _, line := range lines {
				stageFA.Next(line)
				deployFA.Next(line)
			}
			//END

			markedForRemoval := []int{}
			for i := 0; i < len(targets)-1; i++ {
				for j := i + 1; j < len(targets); j++ {
					a := targets[i]
					b := targets[j]
					if a.Org == b.Org && a.Space == b.Space {
						if a.Pushed {
							markedForRemoval = append(markedForRemoval, j)
						} else {
							markedForRemoval = append(markedForRemoval, j)
						}
					}
				}
			}
			for i := 0; i < len(markedForRemoval); i++ {
				markedForRemoval[i] -= i
			}
			for _, i := range markedForRemoval {
				targets = append(targets[:i], targets[i+1:]...)
			}

			id := nt.NewUuid().String()
			temp := t.JenkinsBuildTargets{id, entry.Id, tim, build.Number, sha, targets}
			allShasAdded[sha] = struct{}{}
			//log.Println(sha)
			//log.Printf("%+v\n", temp)
			m.app.index.PostDataWait(JenkinsBuildTargets_QType, id, temp)
			//log.Println(m.index.PostDataWait(m.targetsType, id, temp))
		}
		if len(allShasAdded) != 0 {
			repo, err := m.app.rtrvr.GetRepositoryById(entry.RepositoryId)
			if err == nil {
				for sha, _ := range allShasAdded {
					m.app.ff.FireRequest(&SingleRunnerRequest{repo, sha, ""})
				}
			}
			fmt.Println(allShasAdded)
		}
	}
	return nil
}

func (m *JenkinsManager) GetOrgsAndSpaces(pipelineId string) (map[string][]string, error) {
	q := map[string]interface{}{
		"aggs": map[string]interface{}{
			"targets": map[string]interface{}{
				"nested": map[string]interface{}{"path": t.JenkinsBuildTargets_QField_CFTargets},
				"aggs": map[string]interface{}{
					"orgs": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Org),
							"size":  10000,
						},
						"aggs": map[string]interface{}{
							"spaces": map[string]interface{}{
								"terms": map[string]interface{}{
									"field": t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Space),
									"size":  10000,
								},
							},
						},
					},
				},
			},
		},
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				t.JenkinsBuildTargets_QField_PipelineId: pipelineId,
			},
		},
		"size": 0,
	}
	resp, err := m.app.index.SearchByJSON(JenkinsBuildTargets_QType, q)
	if err != nil {
		return nil, err
	}
	nested, ok := resp.Aggregations.Nested("targets")
	if !ok {
		return nil, u.Error("Query did not return a valid nested section")
	}
	orgsAgg, ok := nested.Terms("orgs")
	if !ok {
		return nil, u.Error("The query did not return a valid org agg")
	}
	res := map[string][]string{}
	for _, org := range orgsAgg.Buckets {
		spacesAgg, ok := org.Terms("spaces")
		if !ok {
			return nil, u.Error("The org agg did not return a valid space agg")
		}
		spaces := make([]string, len(spacesAgg.Buckets), len(spacesAgg.Buckets))
		for i, space := range spacesAgg.Buckets {
			spaces[i] = space.Key.(string)
		}
		res[org.Key.(string)] = spaces
	}
	return res, nil
}

func (m *JenkinsManager) GetLastSuccesses(repoId string) (map[string]map[string]string, error) {
	res := map[string]map[string]string{}
	orgsSpaces, err := m.GetOrgsAndSpaces(repoId)
	if err != nil {
		return nil, err
	}
	for org, spaces := range orgsSpaces {
		res[org] = map[string]string{}
		for _, space := range spaces {
			q := map[string]interface{}{
				"query": es.NewNestedQuery(t.JenkinsBuildTargets_QField_CFTargets).SetInnerQuery(map[string]interface{}{
					"bool": es.NewBool().
						SetMust(es.NewBoolQ(
							es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Org), org),
							es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Space), space),
							es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Pushed), true),
							es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Stage, t.JenkinsBuildStage_QField_Success), true)))}),
				"sort": map[string]interface{}{
					t.JenkinsBuildTargets_QField_Timestamp: "desc",
				},
				"size": 1,
			}
			resp, err := m.app.index.SearchByJSON(JenkinsBuildTargets_QType, q)
			if err != nil {
				return nil, err
			}
			if len(resp.Hits.Hits) < 1 {
				continue
			}
			target := new(t.JenkinsBuildTargets)
			if err = json.Unmarshal([]byte(*resp.Hits.Hits[0].Source), target); err != nil {
				return nil, err
			}
			res[org][space] = target.Sha
		}
	}
	return res, nil
}

func (m *JenkinsManager) GetAllSuccesses(repoId string) (map[string]map[string][]string, error) {
	res := map[string]map[string][]string{}
	orgsSpaces, err := m.GetOrgsAndSpaces(repoId)
	if err != nil {
		return nil, err
	}
	for org, spaces := range orgsSpaces {
		res[org] = map[string][]string{}
		for _, space := range spaces {
			agg := es.NewAggQuery("shas", "sha")
			nested := es.NewNestedQuery(t.JenkinsBuildTargets_QField_CFTargets)
			boool := es.NewBool().SetMust(es.NewBoolQ(
				es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Org), org),
				es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Space), space),
				es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Pushed), true),
				es.NewTerm(t.DotJoin(t.JenkinsBuildTargets_QField_CFTargets, t.CFTarget_QField_Stage, t.JenkinsBuildStage_QField_Success), true)))
			nested.SetInnerQuery(map[string]interface{}{"bool": boool})
			agg["query"] = nested
			resp, err := m.app.index.SearchByJSON(JenkinsBuildTargets_QType, agg)
			res[org][space], err = es.GetAggKeysFromSearchResponse("shas", resp, err)
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}
