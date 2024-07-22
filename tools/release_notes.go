// Copyright 2021 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file  except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the  License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const BASE_URL = "https://issues.couchbase.com"

type Issue struct {
	Key    string      `json:"key"`
	Fields IssueFields `json:"fields"`
}

type IssueFields struct {
	Resolution  IssueResolution `json:"resolution"`
	Status      IssueStatus     `json:"status"`
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	Issuetype   IssueType       `json:"issuetype"`
}

type IssueType struct {
	Name string `json:"name"`
}
type IssueResolution struct {
	Name string `json:"name"`
}

type IssueStatus struct {
	Name string `json:"name"`
}
type GetIssuesResponse struct {
	Issues     []Issue `json:"issues"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
}

type JiraClient struct {
	baseUrl string
	client  *http.Client
}

func (jc *JiraClient) GetIssues(projectName, fixVersion string) ([]Issue, error) {

	reqJson, err := json.Marshal(map[string]any{
		"jql": "fixVersion = " + fixVersion + " AND project=" + projectName + " AND resolution != Unresolved",
	})

	if err != nil {
		return nil, err
	}
	requestBody := bytes.NewBuffer(reqJson)

	request, err := newRequest("POST", jc.baseUrl+"/rest/api/2/search", requestBody)

	if err != nil {
		return nil, err
	}

	res, err := jc.client.Do(request)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	queryResponse, err := ioutil.ReadAll(res.Body)
	if 200 > res.StatusCode && res.StatusCode >= 300 {
		return nil, fmt.Errorf("non 200 range status code: %d", res.StatusCode)
	}

	getRes := &GetIssuesResponse{}
	json.Unmarshal(queryResponse, getRes)
	if err != nil {
		return nil, err
	}

	if getRes.MaxResults == getRes.Total {
		//TODO: implement this
		log.Panic("Need to retrieve more issues")
	}
	return getRes.Issues, nil

}

func newRequest(method string, url string, body *bytes.Buffer) (*http.Request, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	token := os.Getenv("JIRA_TOKEN")
	if token == "" {
		log.Fatal("JIRA_TOKEN not set")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("content-type", "application/json")

	return request, nil
}

func createIssueLink(name string) string {
	return fmt.Sprintf("https://issues.couchbase.com/browse/%s[%s^]", name, name)
}
func write(file io.StringWriter, format string, args ...interface{}) {
	line := fmt.Sprintf(format+"\n", args...)

	if _, err := file.WriteString(line); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func filterIssuesbyType(issues []Issue, predicate ...string) *[]Issue {
	res := []Issue{}
	for _, v := range issues {
		if stringInSlice(v.Fields.Issuetype.Name, predicate) {
			res = append(res, v)
		}
	}
	return &res
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func main() {

	project := "CMOS"
	ver := "0.3"
	jc := JiraClient{
		baseUrl: BASE_URL,
		client:  &http.Client{},
	}
	myRes, err := jc.GetIssues(project, ver)
	if err != nil {
		log.Panic(err)
	}

	file, err := os.OpenFile("docs/modules/ROOT/pages/release-notes.adoc", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}

	write(file, "// THIS FILE IS AUTO-GENERATED - DO NOT EDIT")

	// Headers
	write(file, "= Release Notes")
	write(file, "")
	write(file, "== Installation")
	write(file, "")
	write(file, "For installation instructions, refer to the xref:quickstart.adoc[CMOS Quick Start] page.")
	write(file, "")
	write(file, "== Release %s", ver)
	write(file, "")

	if featIssues := filterIssuesbyType(myRes, "Improvement", "Task", "New Feature", "Sub-task"); len(*featIssues) != 0 {

		write(file, "=== Features")
		for _, v := range *featIssues {
			write(file, "* Issue: %s - `%s`", createIssueLink(v.Key), NormaliseString(v.Fields.Summary))
		}
		write(file, "")
	}
	if bugIssues := filterIssuesbyType(myRes, "Bug"); len(*bugIssues) != 0 {

		write(file, "=== Bug Fixes")
		for _, v := range *bugIssues {
			write(file, "* Issue: %s - `%s`", createIssueLink(v.Key), NormaliseString(v.Fields.Summary))
		}
		write(file, "")
	}

	write(file, "=== Known Issues")
	write(file, "// TODO")
	write(file, "== Feedback")
	write(file, "// TODO")
	write(file, "== Licences for Third-Party Components")
	write(file, "// TODO")
	write(file, "== More Information")
	write(file, "// TODO")
	file.Close()
}

func NormaliseString(s string) string {
	return strings.Trim(s, " ")
}
