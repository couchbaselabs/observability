// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package config

import (
	"fmt"

	"github.com/couchbase/tools-common/cbvalue"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"
)

type Feature string

const (
	FeatHealthAgent        Feature = "health-agent"
	FeatFluentBit          Feature = "fluent-bit"
	FeatLogAnalyzer        Feature = "log-analyzer"
	FeatPrometheusExporter Feature = "prometheus-exporter"
)

func StringToFeature(val string) (Feature, bool) {
	switch Feature(val) {
	case FeatHealthAgent, FeatFluentBit, FeatLogAnalyzer, FeatPrometheusExporter:
		return Feature(val), true
	default:
		return "", false
	}
}

type FeatureSet map[Feature]bool

func (fs FeatureSet) AllDisabled() bool {
	for _, f := range fs {
		if f {
			return false
		}
	}
	return true
}

func NoFeatures() FeatureSet {
	return FeatureSet{
		FeatHealthAgent:        false,
		FeatFluentBit:          false,
		FeatLogAnalyzer:        false,
		FeatPrometheusExporter: false,
	}
}

func AllFeatures() FeatureSet {
	return FeatureSet{
		FeatHealthAgent:        true,
		FeatFluentBit:          true,
		FeatLogAnalyzer:        true,
		FeatPrometheusExporter: true,
	}
}

func AutoFeatures(node *bootstrap.Node) FeatureSet {
	result := NoFeatures()

	result[FeatHealthAgent] = true
	result[FeatFluentBit] = true
	result[FeatLogAnalyzer] = true

	if node.Version().Older(cbvalue.Version7_0_0) {
		result[FeatPrometheusExporter] = true
	}

	return result
}

func DetermineFeaturesBasedOnFlags(defaultOn bool, enable, disable []string) (FeatureSet, error) {
	var features FeatureSet
	if defaultOn {
		features = AllFeatures()
	} else {
		features = NoFeatures()
	}

	for _, featName := range disable {
		feat, ok := StringToFeature(featName)
		if !ok {
			return nil, fmt.Errorf("unknown feature to disable: '%s'", featName)
		}
		features[feat] = false
	}

	for _, featName := range enable {
		feat, ok := StringToFeature(featName)
		if !ok {
			return nil, fmt.Errorf("unknown feature to enable: '%s'", featName)
		}
		features[feat] = true
	}

	return features, nil
}

func DetermineFeaturesForNode(node *bootstrap.Node, auto bool, enable, disable []string) (FeatureSet, error) {
	var features FeatureSet
	if auto {
		features = AutoFeatures(node)
	} else {
		features = NoFeatures()
	}

	for _, featName := range disable {
		feat, ok := StringToFeature(featName)
		if !ok {
			return nil, fmt.Errorf("unknown feature to disable: '%s'", featName)
		}
		features[feat] = false
	}

	for _, featName := range enable {
		feat, ok := StringToFeature(featName)
		if !ok {
			return nil, fmt.Errorf("unknown feature to enable: '%s'", featName)
		}
		features[feat] = true
	}

	return features, nil
}
