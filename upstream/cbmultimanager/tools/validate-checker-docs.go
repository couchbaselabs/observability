// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// documentedCheckerIDRe matches a checker ID (CB12345), preceded either by a hash or a whitespace.
// This is so it matches either AsciiDoc ID syntax (like [#CB12345]), or comments (for checkers documented elsewhere).
var documentedCheckerIDRe = regexp.MustCompile(`[#\s](CB[0-9]+)`)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	defsFile := filepath.Join(cwd, "cluster-monitor", "pkg", "values", "checker_defs.yaml")
	defsFD, err := os.Open(defsFile)
	if err != nil {
		panic(err)
	}
	defer defsFD.Close()

	var knownCheckersByName map[string]values.CheckerDefinition
	if err := yaml.NewDecoder(defsFD).Decode(&knownCheckersByName); err != nil {
		panic(err)
	}

	knownCheckersByID := make(map[string]string, len(knownCheckersByName))
	for _, defn := range knownCheckersByName {
		knownCheckersByID[defn.ID] = defn.Name
	}

	docsPath := filepath.Join(cwd, "docs", "modules", "ROOT", "pages", "checkers.adoc")
	docs, err := os.ReadFile(docsPath)
	if err != nil {
		panic(err)
	}

	documentedCheckers := make(map[string]bool)
	matches := documentedCheckerIDRe.FindAllStringSubmatch(string(docs), -1)
	if matches == nil {
		fmt.Println("Found no documented checkers at all (that can't be right!)")
		os.Exit(1)
	}
	for _, match := range matches {
		documentedCheckers[match[1]] = true
	}

	var problems []string

	for id := range knownCheckersByID {
		if _, ok := documentedCheckers[id]; !ok {
			problems = append(problems, fmt.Sprintf("Checker %s is not documented."+
				"If it's documented in couchbaselabs/observability, add a comment with its ID to checkers.adoc.", id))
		}
	}
	for id := range documentedCheckers {
		if _, ok := knownCheckersByID[id]; !ok {
			problems = append(
				problems,
				fmt.Sprintf("Checker %s is documented but not known.", id),
			)
		}
	}

	if len(problems) == 0 {
		os.Exit(0)
	}
	fmt.Println("Problems: ")
	for _, prob := range problems {
		fmt.Printf("\t%s\n", prob)
	}
	os.Exit(1)
}
