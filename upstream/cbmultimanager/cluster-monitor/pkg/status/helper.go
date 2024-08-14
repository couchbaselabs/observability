// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package status

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

type parsedIndexes struct {
	equivalentIndexes            map[string][]*values.IndexStatus
	indexesWithMissingPartitions []*values.IndexStatus
}

func versionCheck(checkerName string, cluster *values.CouchbaseCluster,
	checkerDefs map[string]values.CheckerDefinition,
) *values.WrappedCheckerResult {
	def, ok := checkerDefs[checkerName]
	if !ok || def.MinVersion == "" {
		return nil
	}

	if cluster.NodesSummary.GetMinVersion().AtLeast(def.MinVersion) {
		return nil
	}

	return &values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name: checkerName,
			Remediation: fmt.Sprintf("Cluster has to be at least %s to run this checker. Upgrade cluster to get more"+
				" checkers.", def.MinVersion),
			Status: values.MissingCheckerStatus,
			Time:   time.Now().UTC(),
		},
		Cluster: cluster.UUID,
	}
}

// indexDefinitionRe will extract to the first sub-match group the key parts of a CREATE INDEX statement that define
// whether two indexes are considered equivalent - everything between the `ON` ( including the keyspace) until either
// the end or the WITH clause.
// GSI will do a certain amount of whitespace/punctuation normalisation internally, so we don't have to
// worry about that.
// Reference: https://docs.couchbase.com/server/current/n1ql/n1ql-language-reference/createindex.html#syntax
var (
	indexDefinitionRe = regexp.MustCompile(`ON (.*?)\s*(?:WITH .*)?\s*$`)
	numPartitionRe    = regexp.MustCompile(`WITH.*"num_partition":\s*?([0-9]+)`)
)

// parseIndexes completes any functions that require all indexes to be checked,
// this means we do not have to loop over these indexes multiple times.
func parseIndexes(indexes []*values.IndexStatus) (*parsedIndexes, error) {
	parsedIndexes := parsedIndexes{equivalentIndexes: make(map[string][]*values.IndexStatus)}
	for _, idx := range indexes {
		defn := groupEquivalentIndex(idx)
		parsedIndexes.equivalentIndexes[defn] = append(parsedIndexes.equivalentIndexes[defn], idx)
		definitionPartitions := numPartitionRe.FindStringSubmatch(idx.Definition)
		if definitionPartitions != nil {
			missing, err := indexDefnPartitions(idx, definitionPartitions)
			if err != nil {
				return nil, err
			}
			if missing {
				parsedIndexes.indexesWithMissingPartitions = append(parsedIndexes.indexesWithMissingPartitions, idx)
			}
		}
	}
	return &parsedIndexes, nil
}

// groupEquivalentIndexes groups together all equivalent indexes - this includes replicas as well as indexes with a
// different name but the same condition.
func groupEquivalentIndex(index *values.IndexStatus) string {
	defnMatch := indexDefinitionRe.FindStringSubmatch(index.Definition)
	var defn string
	if defnMatch != nil {
		defn = defnMatch[1]
	} else {
		zap.S().Warnw("indexDefinitionRe failed to match",
			"definition", index.Definition)
	}
	return defn
}

// indexDefnPartitions gets the number of partitions declared in the original index definition.
// It doesn't use NumPartitions as that only shows active partitions, not missing ones.
func indexDefnPartitions(index *values.IndexStatus, defnPartitions []string) (bool, error) {
	defn := defnPartitions[1]
	defPartitionsInt, err := strconv.Atoi(defn)
	if err != nil {
		return false, fmt.Errorf("could not convert numPartition string to int: %w", err)
	}
	var actualPartitions int
	for _, parts := range index.PartitionMap {
		actualPartitions = actualPartitions + len(parts)
	}
	return defPartitionsInt != actualPartitions, nil
}

// makeIndexCheckerResult is a helper for indexesChecks to assemble a WrappedCheckerResult from a list of indexes
// that violate the given checker.
func makeIndexCheckerResult(checkerName string, badIndexes []*values.IndexStatus, cluster values.CouchbaseCluster,
	badIndexStatus values.CheckerStatus, remediation string,
) *values.WrappedCheckerResult {
	result := values.WrappedCheckerResult{
		Result: &values.CheckerResult{
			Name:   checkerName,
			Status: values.GoodCheckerStatus,
			Time:   time.Now(),
		},
		Cluster: cluster.UUID,
	}
	if len(badIndexes) == 0 {
		return &result
	}

	result.Result.Status = badIndexStatus

	idxNames := make([]string, len(badIndexes))
	for i, idx := range badIndexes {
		idxNames[i] = idx.IndexName
	}
	idxNamesArray, err := json.Marshal(idxNames)
	if err == nil {
		result.Result.Value = idxNamesArray
	} else {
		result.Result.Status = values.MissingCheckerStatus
		result.Error = err
	}
	result.Result.Remediation = remediation
	return &result
}

// fragCalculator calculates the percentage of a node's allocated memory that is fragmented
// and whether that percentage is higher than passed percentages
func fragCalculator(fragmentation string,
	heap string,
	warnPercent float64,
	alertPercent float64,
) (float64, values.CheckerStatus, error) {
	severity := values.GoodCheckerStatus
	frag, err := strconv.Atoi(fragmentation)
	if err != nil {
		return 0, values.GoodCheckerStatus, fmt.Errorf("could not parse FragmentationBytes: %w", err)
	}
	resident, err := strconv.Atoi(heap)
	if err != nil {
		return 0, values.GoodCheckerStatus, fmt.Errorf("could not parse ResidentBytes: %w", err)
	}
	fragPercent := float64(frag) * 100.0 / float64(resident)
	switch {
	case fragPercent > alertPercent:
		severity = values.AlertCheckerStatus
	case fragPercent > warnPercent:
		severity = values.WarnCheckerStatus
	}
	return fragPercent, severity, nil
}
