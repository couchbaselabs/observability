// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

// Package prometheus provides a cluster discovery system based on querying Prometheus for Couchbase Server targets.
package prometheus

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

const (
	DefaultCouchbaseManagementInsecurePort = 8091
	DefaultCouchbaseManagementSecurePort   = 18091
)

const AddressLabel = "__address__"

const (
	ProtocolInsecure = "couchbase"
	ProtocolSecure   = "couchbases"
)

//go:generate mockery --name promAPI --exported

// promAPI is a subset of promv1.API, meant to facilitate mocking for tests.
type promAPI interface {
	Targets(ctx context.Context) (promv1.TargetsResult, error)
}

type CouchbaseClusterDiscovery struct {
	cfg   *configuration.Config
	store storage.Store
	prom  promAPI
}

func NewPrometheusCouchbaseClusterDiscovery(cfg *configuration.Config, store storage.Store) (
	*CouchbaseClusterDiscovery, error,
) {
	client, err := promapi.NewClient(promapi.Config{
		Address: cfg.PrometheusBaseURL,
	})
	if err != nil {
		return nil, err
	}
	return &CouchbaseClusterDiscovery{
		cfg:   cfg,
		store: store,
		prom:  promv1.NewAPI(client),
	}, nil
}

func (p *CouchbaseClusterDiscovery) Discover(ctx context.Context) error {
	targets, err := p.prom.Targets(ctx)
	if err != nil {
		return err
	}

	seenClusters := make(map[string]bool)
	for _, tgt := range targets.Active {
		if !p.matchesLabelSelector(tgt) {
			continue
		}
		cb, address, err := p.findManagementAddressForCluster(tgt.DiscoveredLabels[AddressLabel])
		if err != nil {
			zap.S().Warnw("(Prometheus Discovery) Failed to connect to target", "target", tgt, "err", err)
			continue
		}

		cluster := cb.GetClusterInfo()
		uuid := cluster.ClusterUUID
		if _, seen := seenClusters[uuid]; seen {
			zap.S().Debugw("(Prometheus Discovery) Already processed this cluster", "cluster", uuid)
			continue
		}

		_, err = p.store.GetCluster(uuid, false)
		if err == nil {
			zap.S().Debugw("(Prometheus Discovery) Already got this cluster stored", "cluster", uuid)
			seenClusters[uuid] = true
			continue
		} else if errors.Is(err, values.ErrNotFound) {
			buckets, err := cb.GetBucketsSummary()
			if err != nil {
				zap.S().Warnw("(Prometheus Discovery) Connected to cluster, but could not get buckets summary", "err", err)
				continue
			}

			zap.S().Infow("(Prometheus Discovery) Discovered new cluster", "cluster", uuid, "address", address)

			clusterInfo := values.CouchbaseCluster{
				UUID:           uuid,
				Name:           cluster.ClusterName,
				Enterprise:     cluster.Enterprise,
				ClusterInfo:    cluster.ClusterInfo,
				NodesSummary:   cluster.NodesSummary,
				Alias:          "",
				User:           p.cfg.CouchbaseUser,
				Password:       p.cfg.CouchbasePassword,
				HeartBeatIssue: values.NoHeartIssue,
				BucketsSummary: buckets,
			}
			if err = p.store.AddCluster(&clusterInfo); err != nil {
				return fmt.Errorf("failed to store new cluster: %w", err)
			}
			seenClusters[uuid] = true
		} else {
			return fmt.Errorf("failed to check existing cluster in store for UUID %v: %w", uuid, err)
		}
	}
	// Now check for disappeared clusters
	clusters, err := p.store.GetClusters(false, false)
	if err != nil {
		return fmt.Errorf("couldn't get clusters: %w", err)
	}

	for _, cluster := range clusters {
		if _, seen := seenClusters[cluster.UUID]; seen {
			continue
		}
		zap.S().Infow("(Prometheus Discovery) Cluster in our storage not found in Prometheus, removing",
			"cluster", cluster.UUID)
		if err = p.store.DeleteCluster(cluster.UUID); err != nil {
			return fmt.Errorf("failed to delete cluster %s: %w", cluster.UUID, err)
		}
	}
	return nil
}

func (p *CouchbaseClusterDiscovery) findManagementAddressForCluster(
	knownAddress string,
) (*couchbase.Client, string, error) {
	// TODO (CMOS-101): this is a fairly expensive operation, involving three failing REST calls in the worst case,
	// plus however many retries cbrest performs.
	// It'd be good if we could first check if the host exists in a known cluster, rather than trying and failing to get
	// its cluster UUID.
	// Failing that, we could keep a cache of nodes we couldn't get anything from, to avoid retrying them over and
	// over again. (Purge it occasionally just in case they've come back.)

	host, port := getHostAndPort(knownAddress)

	// First, try with what we have - if this is a CB7 node it'll Just Work
	var protocol string
	if port == strconv.Itoa(DefaultCouchbaseManagementSecurePort) {
		protocol = ProtocolSecure
	} else {
		protocol = ProtocolInsecure
	}
	cb, err := couchbase.NewClient([]string{fmt.Sprintf("%s://%s", protocol, knownAddress)}, p.cfg.CouchbaseUser,
		p.cfg.CouchbasePassword, nil, true)
	if err == nil {
		return cb, knownAddress, nil
	}

	// Try replacing the port with the secure management port
	cb, err = couchbase.NewClient([]string{
		fmt.Sprintf("%s://%s:%d", ProtocolSecure, host,
			DefaultCouchbaseManagementSecurePort),
	}, p.cfg.CouchbaseUser, p.cfg.CouchbasePassword, nil, true)
	if err == nil {
		return cb, fmt.Sprintf("%s:%d", host, DefaultCouchbaseManagementSecurePort), nil
	}

	// Ditto for the insecure port
	cb, err = couchbase.NewClient([]string{
		fmt.Sprintf("%s://%s:%d", ProtocolInsecure, host,
			DefaultCouchbaseManagementInsecurePort),
	}, p.cfg.CouchbaseUser, p.cfg.CouchbasePassword, nil, true)
	if err == nil {
		return cb, fmt.Sprintf("%s:%d", host, DefaultCouchbaseManagementInsecurePort), nil
	}

	// TODO (CMOS-91): This doesn't support custom ports.
	// Bail out.
	return nil, "", fmt.Errorf("could not create cluster client; last error: %w ("+
		"note that non-default Couchbase Server ports are not yet supported)", err)
}

func (p *CouchbaseClusterDiscovery) matchesLabelSelector(target promv1.ActiveTarget) bool {
	for key, val := range p.cfg.PrometheusLabelSelector {
		itsVal, ok := target.Labels[model.LabelName(key)]
		if !ok {
			return false
		}
		if string(itsVal) != val {
			return false
		}
	}
	return true
}

func getHostAndPort(host string) (string, string) {
	ip4Regex := regexp.MustCompile(`^(((\d{1,2})|` +
		`(1\d{1,2})|(2[0-4]\d)|(25[0-5]))\.){3}` +
		`((\d{1,2})|(1\d{1,2})|(2[0-4]\d)|(25[0-5]))` +
		`(:((\d|[1-9]\d{1,3}|` +
		`[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])))?$`)

	port := ""

	if ip4Regex.MatchString(host) {
		if strings.Contains(host, ":") {
			port = host[strings.LastIndex(host, ":")+1:]
			host = host[:strings.LastIndex(host, ":")]
			return host, port
		}
	}

	// ipv6 regex avoided for flakiness and readability concerns
	// instead net.ParseIP and To16 used.
	if strings.Count(host, ":") > 1 {
		if strings.Contains(host, "[") && strings.Contains(host, "]") {
			// check port
			if strings.Contains(host, "]:") {
				port = host[strings.Index(host, "]:")+2:]
			}

			host = host[strings.Index(host, "[")+1 : strings.Index(host, "]")]
			return host, port
		}

		// checks if not a valid representation of an IPv6 address
		if net.ParseIP(host).To16() == nil {
			return host, port
		}
	}

	return host, port
}
