// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package prometheus

import (
	"fmt"
	"io"

	"github.com/couchbaselabs/cbmultimanager/agent/pkg/bootstrap"

	"github.com/couchbase/tools-common/cbrest"
	"github.com/couchbaselabs/cmos-prometheus-exporter/pkg/metrics"
	"github.com/couchbaselabs/cmos-prometheus-exporter/pkg/metrics/eventing"
	"github.com/couchbaselabs/cmos-prometheus-exporter/pkg/metrics/fts"
	"github.com/couchbaselabs/cmos-prometheus-exporter/pkg/metrics/gsi"
	"github.com/couchbaselabs/cmos-prometheus-exporter/pkg/metrics/memcached"
	"github.com/couchbaselabs/cmos-prometheus-exporter/pkg/metrics/n1ql"
	"github.com/couchbaselabs/cmos-prometheus-exporter/pkg/metrics/system"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Exporter struct {
	reg        *prometheus.Registry
	collectors map[cbrest.Service]prometheus.Collector
	// can be nil if the agent isn't ready
	node *bootstrap.Node
	ms   *metrics.MetricSet
}

func must(b bool, err error) bool {
	if err != nil {
		panic(err)
	}
	return b
}

func NewExporter(node *bootstrap.Node) (*Exporter, error) {
	ms := metrics.LoadDefaultMetricSet()
	ex := &Exporter{
		reg:        prometheus.NewPedanticRegistry(),
		collectors: make(map[cbrest.Service]prometheus.Collector),
		node:       node,
		ms:         ms,
	}

	err := ex.reg.Register(prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "cbhealthagent",
		Subsystem: "core",
		Name:      "ready",
		Help:      "1 if the agent is connected to a Couchbase Server and ready to serve requests, 0 otherwise.",
	}, func() float64 {
		if ex.node == nil {
			return 0
		}
		return 1
	}))
	if err != nil {
		return nil, fmt.Errorf("couldn't register core_ready: %w", err)
	}

	if node != nil {
		err = ex.registerServices()
	}

	return ex, err
}

func (p *Exporter) registerServices() error {
	if err := p.registerCollector(cbrest.ServiceData, func() (prometheus.Collector, error) {
		return memcached.NewMemcachedMetrics(zap.S().Named("Prometheus.memcached").Desugar(), p.node, p.ms.Memcached)
	}); err != nil {
		return err
	}

	if err := p.registerCollector(cbrest.ServiceGSI, func() (prometheus.Collector, error) {
		return gsi.NewMetrics(zap.S().Named("Prometheus.GSI"), p.node, p.ms.GSI, false)
	}); err != nil {
		return err
	}

	if err := p.registerCollector(cbrest.ServiceQuery, func() (prometheus.Collector, error) {
		return n1ql.NewMetrics(zap.S().Named("Prometheus.N1QL"), p.node, p.ms.N1QL)
	}); err != nil {
		return err
	}

	if err := p.registerCollector(cbrest.ServiceSearch, func() (prometheus.Collector, error) {
		return fts.NewCollector(zap.S().Named("Prometheus.FTS"), p.node, p.ms.FTS, false), nil
	}); err != nil {
		return err
	}

	if err := p.registerCollector(cbrest.ServiceEventing, func() (prometheus.Collector, error) {
		return eventing.NewCollector(zap.S().Named("Prometheus.Eventing"), p.node, p.ms.Eventing)
	}); err != nil {
		return err
	}

	if err := p.registerCollector(cbrest.ServiceManagement, func() (prometheus.Collector, error) {
		return system.NewSystemMetrics(zap.S().Named("Prometheus.System"), p.ms.System), nil
	}); err != nil {
		return err
	}

	return nil
}

func (p *Exporter) registerCollector(service cbrest.Service, factory func() (prometheus.Collector, error)) error {
	if _, ok := p.collectors[service]; ok {
		return nil
	}
	if !must(p.node.HasService(service)) {
		return nil
	}
	newColl, err := factory()
	if err != nil {
		return fmt.Errorf("couldn't create metrics for %s: %w", service, err)
	}
	err = p.reg.Register(newColl)
	if err != nil {
		return fmt.Errorf("couldn't register metrics for %s: %w", service, err)
	}
	p.collectors[service] = newColl
	return nil
}

func (p *Exporter) Register(router *mux.Router) {
	router.Path("/metrics").Handler(promhttp.HandlerFor(p.reg, promhttp.HandlerOpts{}))
}

func (p *Exporter) Shutdown() {
	for _, coll := range p.collectors {
		if closer, ok := coll.(io.Closer); ok {
			closer.Close()
		}
	}
}
