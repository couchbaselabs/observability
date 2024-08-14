// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

//go:build gen
// +build gen

package main

import (
	_ "embed"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

//go:embed checkers.adoc.tmpl
var templateString string

//go:generate go run -tags gen checkers_gen.go

func main() {
	dataRaw := values.LoadCheckerDefsWithDocs()
	dataSorted := make([]values.CheckerDefWithDoc, 0, len(dataRaw))
	for _, val := range dataRaw {
		dataSorted = append(dataSorted, val)
	}
	sort.Slice(dataSorted, func(i, j int) bool {
		return strings.Compare(dataSorted[i].ID, dataSorted[j].ID) == -1
	})

	tmpl := template.New("checkers.adoc")
	template.Must(tmpl.Parse(templateString))

	out, err := os.OpenFile("checkers.adoc", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		panic(err)
	}
	defer out.Close()
	if err := tmpl.Execute(out, dataSorted); err != nil {
		panic(err)
	}
}
