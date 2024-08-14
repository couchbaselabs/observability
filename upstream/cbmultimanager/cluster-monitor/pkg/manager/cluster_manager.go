// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package manager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/couchbase/tools-common/aprov"
	"github.com/couchbase/tools-common/errdefs"
	"github.com/couchbase/tools-common/netutil"
	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/agentport"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/configuration/tunables"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/meta"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/statistics"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/alertmanager"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// CMOS-267 tracks making this configurable.
const standaloneAgentPort = 9092

type SingleClusterManager struct {
	clusterLock sync.RWMutex
	cluster     *values.CouchbaseCluster
	client      *couchbase.Client

	store         storage.Store
	alertmanager  *alertmanager.AlertGenerator
	checkExecutor *status.CheckExecutor

	agentPortLock sync.RWMutex
	agentPorts    map[string]*agentport.AgentPort

	frequencies values.FrequencyConfiguration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	logger *zap.SugaredLogger

	isRunning bool
}

func NewSingleClusterManager(
	cluster *values.CouchbaseCluster,
	client *couchbase.Client,
	store storage.Store,
	am *alertmanager.AlertGenerator,
	checkExecutor *status.CheckExecutor,
	freqs values.FrequencyConfiguration,
) *SingleClusterManager {
	return &SingleClusterManager{
		logger:        zap.S().Named("Single Cluster Manager").With("cluster", cluster.UUID),
		cluster:       cluster,
		client:        client,
		store:         store,
		alertmanager:  am,
		checkExecutor: checkExecutor,
		frequencies:   freqs,
		agentPorts:    make(map[string]*agentport.AgentPort),
	}
}

func (s *SingleClusterManager) Start() error {
	if s.isRunning {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	// Note: setting up agent ports can take some time, so it runs in the background to avoid blocking manager startup,
	// however this means that the agent ports may not be in place in time for the first checker run.
	go s.agentPortReconcileLoop()
	go s.heartLoop()
	if s.cluster.Enterprise {
		go s.checkerLoop()
	}
	s.isRunning = true

	return nil
}

func (s *SingleClusterManager) Stop() {
	if !s.isRunning {
		return
	}

	s.cancel()
	s.wg.Wait()

	s.isRunning = false
}

func (s *SingleClusterManager) UpdateClusterInfo(cluster *values.CouchbaseCluster) {
	s.clusterLock.Lock()
	s.cluster = cluster
	s.clusterLock.Unlock()
}

func (s *SingleClusterManager) GetProgress() (*values.ClusterProgress, error) {
	return s.checkExecutor.GetProgressFor(s.cluster.UUID)
}

func (s *SingleClusterManager) ManuallyRunCheckers() error {
	if !s.cluster.Enterprise {
		s.logger.Info("Cluster is not enterprise, not running checkers.")
		return nil
	}
	s.logger.Info("Manually running checkers.")
	s.runCheckers()
	s.updateAgentCheckers()
	return nil
}

func (s *SingleClusterManager) ManuallyHeartBeat() error {
	s.logger.Info("Manually heart-beating cluster")
	return s.heartbeat()
}

func (s *SingleClusterManager) heartLoop() {
	err := s.heartbeat()
	if err != nil {
		s.logger.Errorw("Failed to heartbeat", "error", err)
	}
	timer := time.NewTimer(s.frequencies.Heart)
	for {
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := s.heartbeat(); err != nil {
				s.logger.Errorw("Failed to heartbeat", "error", err)
			}
			timer.Reset(s.frequencies.Heart)
		}
	}
}

func (s *SingleClusterManager) checkerLoop() {
	s.runCheckers()
	s.updateAgentCheckers()
	timer := time.NewTimer(s.frequencies.Status)
	for {
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.runCheckers()
			s.updateAgentCheckers()
			timer.Reset(s.frequencies.Status)
		}
	}
}

func (s *SingleClusterManager) agentPortReconcileLoop() {
	s.reconcileAgentPorts()
	timer := time.NewTimer(s.frequencies.AgentPortReconcile)
	for {
		select {
		case <-s.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			s.reconcileAgentPorts()
			timer.Reset(s.frequencies.AgentPortReconcile)
		}
	}
}

func (s *SingleClusterManager) runCheckers() {
	start := time.Now()
	defer func() {
		s.logger.Debugw("Checker run complete.", "elapsed", time.Since(start))
	}()
	s.wg.Add(1)
	defer s.wg.Done()

	s.logger.Debugw("Starting checker run")

	s.clusterLock.Lock()
	cluster := *s.cluster
	s.clusterLock.Unlock()

	if s.client != nil {
		s.client.Close()
	}

	var err error
	s.client, err = couchbase.NewClient(cluster.NodesSummary.GetHosts(), cluster.User,
		cluster.Password, cluster.GetTLSConfig(), false)
	if err != nil {
		s.logger.Errorw("unable to establish connection with cluster", "error", err)
		// adding fast fail to avoid seg faults when unable to communicate with cluster.
		return
	}
	err = s.updateClusterData()
	if err != nil {
		s.logger.Errorw("unable to update cluster", "error", err)
		return
	}

	s.clusterLock.Lock()
	cluster = *s.cluster
	s.clusterLock.Unlock()

	// once we have the copy of cluster
	// make api calls to populate cache data
	cluster.CacheRESTData = *s.retrieveCheckersData(cluster.CacheRESTData,
		cluster.CacheRESTDataErrors, cluster.BucketsSummary)

	// CheckExecutor is responsible for updating progress
	resultsCh, err := s.checkExecutor.CheckCluster(cluster)
	if err != nil {
		s.logger.Errorw("Failed to schedule cluster check", "error", err)
	}
	results := <-resultsCh

	s.applyCheckerResults(results)
}

func (s *SingleClusterManager) retrieveCheckersData(clusterCacheRESTData values.CacheRESTData,
	clusterCacheRESTDataErrors values.CacheRESTDataErrors,
	bucketsSummary values.BucketsSummary,
) *values.CacheRESTData {
	// make REST API calls

	// get pools/default/buckets data
	buckets, err := s.client.GetPoolsBucket()
	if err != nil {
		s.logger.Errorw("Failed to get pools/default/buckets: %w", err)
		clusterCacheRESTDataErrors.BucketsError = err
	}

	// get every bucket stats
	bucketsStats := make([]*values.BucketStat, 0)
	bucketsStatsErrors := make(map[string]error)
	for _, bucket := range bucketsSummary {
		bucketStat, err := s.client.GetBucketStats(bucket.Name)
		bucketsStatsErrors[bucket.Name] = nil
		if err != nil {
			s.logger.Errorw("Failed to get bucket "+bucket.Name+": %w", err)
			bucketsStatsErrors[bucket.Name] = err
		}
		bucketsStats = append(bucketsStats, bucketStat)
	}
	clusterCacheRESTDataErrors.BucketStatsErrors = bucketsStatsErrors

	// get /pools/default/serverGroups data
	serverGroup, err := s.client.GetServerGroups()
	if err != nil {
		s.logger.Errorw("Failed to get /pools/default/serverGroups: %w", err)
		clusterCacheRESTDataErrors.ServerGroupsError = err
	}

	// get /nodes/self
	nodeStorage, err := s.client.GetNodeStorage()
	if err != nil {
		s.logger.Errorw("Failed to get /nodes/self: %w", err)
		clusterCacheRESTDataErrors.NodeStorageError = err
	}

	// get /settings/autoFailover
	autoFailOverSettings, err := s.client.GetAutoFailOverSettings()
	if err != nil {
		s.logger.Errorw("Failed to get /settings/autoFailover: %w", err)
		clusterCacheRESTDataErrors.AutoFailoverSettingsError = err
	}

	// get /indexStatus
	indexStatus, err := s.client.GetIndexStatus()
	if err != nil {
		s.logger.Errorw("Failed to get /indexStatus: %w", err)
		clusterCacheRESTDataErrors.IndexStatusError = err
	}

	// get /settings/indexes
	gsiSettings, err := s.client.GetGSISettings()
	if err != nil {
		s.logger.Errorw("Failed to get /settings/indexes: %w", err)
		clusterCacheRESTDataErrors.GSISettingsError = err
	}

	// get index storage stats
	indexStorageStats, err := s.client.GetIndexStorageStats()
	if err != nil {
		s.logger.Errorw("Failed to get index storage stats: %w", err)
		clusterCacheRESTDataErrors.IndexStorageStatsError = err
	}

	// get FTS Index Status
	ftsIndexStatus, err := s.client.GetFTSIndexStatus()
	if err != nil {
		s.logger.Errorw("Failed to get FTS index status: %w", err)
		clusterCacheRESTDataErrors.FTSIndexStatusError = err
	}

	// get "analytics/node/diagnostics
	analyticalNodeDiag, err := s.client.GetAnalyticsNodeDiagnostics()
	if err != nil {
		s.logger.Errorw("Failed to get /analytics/node/diagnostics: %w", err)
		clusterCacheRESTDataErrors.AnalyticNodeDiagError = err
	}

	// get UI logs
	uiLogs, err := s.client.GetUILogs()
	if err != nil {
		s.logger.Errorw("Failed to get UI Logs: %w", err)
		clusterCacheRESTDataErrors.UILogsError = err
	}

	return clusterCacheRESTData.WithCacheRESTData(
		buckets,
		bucketsStats,
		serverGroup,
		nodeStorage,
		autoFailOverSettings,
		indexStatus,
		gsiSettings,
		indexStorageStats,
		ftsIndexStatus,
		analyticalNodeDiag,
		uiLogs,
	)
}

func (s *SingleClusterManager) updateAgentCheckers() {
	results, err := s.getAgentCheckers()
	if err != nil {
		s.logger.Errorw("Failed to update agent checkers for some nodes", "error", err)
	}
	if len(results) <= 0 {
		return
	}
	s.applyCheckerResults(results)
}

// getAgentCheckers retrieves checker results from all the agent ports that this SingleClusterManager currently has
// active. Note that getAgentCheckers may return both results and an error, if some agents were able to provide results
// but others were not. In this case, the error will be of type (*errdefs.MultiError).
func (s *SingleClusterManager) getAgentCheckers() ([]*values.WrappedCheckerResult, error) {
	start := time.Now()
	defer func() {
		s.logger.Debugw("Agent checkers acquired.", "elapsed", time.Since(start))
	}()
	s.logger.Debug("Requesting agent checkers")

	nodeResults := make(chan []*values.WrappedCheckerResult, len(s.agentPorts))
	s.agentPortLock.RLock()
	agentPorts := s.agentPorts
	s.agentPortLock.RUnlock()
	nodeErrors := make(chan error, len(s.agentPorts))

	wg := sync.WaitGroup{}

	for host, port := range agentPorts {
		wg.Add(1)
		go func(host string, port *agentport.AgentPort) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(s.ctx, tunables.ClusterManagerAgentCheckerTimeout)
			defer cancel()

			rawResults, err := port.GetCheckerResults(ctx)
			if err != nil {
				nodeErrors <- fmt.Errorf("failed to get checker results from %s: %w", host, err)
				return
			}
			results := make([]*values.WrappedCheckerResult, 0, len(rawResults))
			for key := range rawResults {
				result := rawResults[key]
				results = append(results, &result)
			}
			nodeResults <- results
		}(host, port)
	}

	wg.Wait()
	close(nodeResults)
	close(nodeErrors)

	result := make([]*values.WrappedCheckerResult, 0)
	me := &errdefs.MultiError{
		Prefix: "failed to get checker results from one or more node: ",
	}
	for res := range nodeResults {
		result = append(result, res...)
	}
	for err := range nodeErrors {
		me.Add(err)
	}
	return result, me.ErrOrNil()
}

// applyCheckerResults writes the given results to the store and updates Prometheus and Alertmanager.
func (s *SingleClusterManager) applyCheckerResults(results []*values.WrappedCheckerResult) {
	for _, result := range results {
		if result.Error != nil {
			s.logger.Errorw("Encountered error on checker function",
				"error", result.Error, "checkerName", getCheckerName(result))
			continue
		}
		if err := s.store.SetCheckerResult(result); err != nil {
			s.logger.Errorw("Failed to store checker result", "checker", getCheckerName(result), "error", err)
		}
	}
	statistics.CheckStatus(results, s.cluster)
	if s.alertmanager != nil {
		_ = s.alertmanager.ManualUpdate(s.ctx)
	}
}

func (s *SingleClusterManager) updateClusterData() error {
	start := time.Now()
	defer func() {
		s.logger.Debugw("Heart beat complete", "elapsed", time.Since(start))
	}()
	s.wg.Add(1)
	defer s.wg.Done()

	if s.client.ClusterInfo.ClusterUUID != s.cluster.UUID {
		s.logger.Warnw("Cluster UUID changed", "old", s.cluster.UUID, "new",
			s.client.ClusterInfo.ClusterUUID)
		s.clusterLock.Lock()
		s.cluster.Enterprise = s.client.ClusterInfo.Enterprise
		s.clusterLock.Unlock()
		if err := s.store.UpdateCluster(&values.CouchbaseCluster{
			UUID:           s.cluster.UUID,
			HeartBeatIssue: values.UUIDMismatchHeartIssue,
			Enterprise:     s.client.ClusterInfo.Enterprise,
		}); err != nil {
			return err
		}
		return fmt.Errorf("cluster uuid has changed from %v to %v -> reinitialise cluster", s.cluster.UUID,
			s.client.ClusterInfo.ClusterUUID)
	}

	remoteClusters, err := s.client.GetRemoteClusters(s.client.ClusterInfo.ClusterName, s.client.ClusterInfo.ClusterUUID)
	if err != nil {
		zap.S().Errorw("(Heart Monitor) Could not update remote clusters", "cluster", s.cluster.UUID, "err", err)
	}

	// Get up to date buckets information. If this fails, still update all the other data
	var buckets values.BucketsSummary
	buckets, err = s.client.GetBucketsSummary()
	if err != nil {
		s.logger.Errorw("Could not update buckets summary", "cluster", s.cluster.UUID, "err", err)
	}

	if err := s.store.UpdateCluster(&values.CouchbaseCluster{
		UUID:           s.cluster.UUID,
		Enterprise:     s.client.ClusterInfo.Enterprise,
		NodesSummary:   s.client.ClusterInfo.NodesSummary,
		Name:           s.client.ClusterInfo.ClusterName,
		ClusterInfo:    s.client.ClusterInfo.ClusterInfo,
		HeartBeatIssue: values.NoHeartIssue,
		BucketsSummary: buckets,
		RemoteClusters: remoteClusters,
		LastUpdate:     start,
	}); err != nil {
		return err
	}
	s.clusterLock.Lock()
	defer s.clusterLock.Unlock()
	s.cluster.LastUpdate = start
	s.cluster.Enterprise = s.client.ClusterInfo.Enterprise
	s.cluster.NodesSummary = s.client.ClusterInfo.NodesSummary
	s.cluster.Name = s.client.ClusterInfo.ClusterName
	s.cluster.ClusterInfo = s.client.ClusterInfo.ClusterInfo
	s.cluster.BucketsSummary = buckets
	s.cluster.PoolsRaw = s.client.ClusterInfo.PoolsRaw
	return nil
}

func (s *SingleClusterManager) heartbeat() error {
	start := time.Now()
	defer func() {
		s.logger.Debugw("Heart beat complete", "elapsed", time.Since(start))
	}()
	s.wg.Add(1)
	defer s.wg.Done()

	// This function can take a while if there are errors, so be careful about acquiring and releasing s.mux.
	// GetHosts returns a new slice, so mux can be safely unlocked after calling it.
	s.clusterLock.RLock()
	hosts := s.cluster.NodesSummary.GetHosts()
	s.clusterLock.RUnlock()

	s.logger.Debugw("Heart beat for cluster", "hosts", hosts)

	if s.client != nil {
		err := s.client.PingNodes()
		// in failure cases update cluster entry to reflect issue
		if err != nil {
			s.logger.Warnw("Cluster heartbeat failed", "err", err)
			issue := values.NoConnectionHeartIssue
			var authError couchbase.AuthError
			if errors.As(err, &authError) {
				issue = values.BadAuthHeartIssue
			}

			s.logger.Info("Refreshing cluster details within 5 minutes. Refresh cluster manually for doing so immediately.")

			// Not taking s.mux as only the store is updated, not the SingleClusterManager state.
			return s.store.UpdateCluster(&values.CouchbaseCluster{
				UUID:           s.cluster.UUID,
				HeartBeatIssue: issue,
			})
		}
	}
	return nil
}

// reconcileAgentPorts ensures that all nodes in this cluster's node summary have agent ports.
func (s *SingleClusterManager) reconcileAgentPorts() {
	start := time.Now()
	s.logger.Debugw("Starting agent port reconcile.")
	defer func() {
		s.logger.Debugw("Agent ports reconciled.", "elapsed", time.Since(start))
	}()
	s.wg.Add(1)
	defer s.wg.Done()

	seen := make(map[string]struct{})

	// Make a copy of the nodes summary to avoid needing to hold s.mux for longer than necessary.
	// NodesSummary is a slice of NodeSummary struct values, so one level of copying is sufficient.
	s.clusterLock.Lock()
	nodes := make(values.NodesSummary, len(s.cluster.NodesSummary))
	copy(nodes, s.cluster.NodesSummary)
	s.clusterLock.Unlock()

	wg := sync.WaitGroup{}

	// Create the agent port in the background as it can block for a while if the agent is just not running.
	for _, node := range nodes {
		seen[node.Host] = struct{}{}
		if _, ok := s.agentPorts[node.Host]; ok {
			continue
		}
		wg.Add(1)
		go func(node values.NodeSummary) {
			defer wg.Done()
			host, _, _ := net.SplitHostPort(netutil.TrimSchema(node.Host))
			ap, err := agentport.NewAgentPort(host, standaloneAgentPort, &aprov.Static{
				UserAgent: fmt.Sprintf("cbmultimanger/%s", meta.Version),
				Username:  s.cluster.User,
				Password:  s.cluster.Password,
			})
			if err != nil {
				s.logger.Errorw("Failed to initialise agent port, will retry", "host", host,
					"error", err)
				return
			}
			s.logger.Debugw("Created agent port.", "host", host)
			s.agentPortLock.Lock()
			s.agentPorts[node.Host] = ap
			s.agentPortLock.Unlock()
		}(node)
	}

	wg.Wait()

	// Now shut down the agent ports of all the nodes that have been removed
	s.agentPortLock.Lock()
	for host, ap := range s.agentPorts {
		if _, ok := seen[host]; ok {
			continue
		}
		go func(ap *agentport.AgentPort) {
			if err := ap.Close(); err != nil {
				s.logger.Warnw("Error shutting down agent port", "host", host, "error", err)
				return
			}
			s.logger.Debugw("Shut down agent port.", "host", host)
		}(ap)
		delete(s.agentPorts, host)
	}
	s.agentPortLock.Unlock()
}

func getCheckerName(results *values.WrappedCheckerResult) string {
	if results.Result == nil {
		return "N/A"
	}

	return results.Result.Name
}
