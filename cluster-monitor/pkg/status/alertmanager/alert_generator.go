// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package alertmanager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/status/alertmanager/types"

	"go.uber.org/zap"

	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/storage"
	"github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

//go:generate mockery --name clock --exported

// clock is an interface to allow manipulating the time in tests.
type clock interface {
	Now() time.Time
}

type wallClock struct{}

func (r *wallClock) Now() time.Time {
	return time.Now()
}

type AlertGenerator struct {
	store          storage.Store
	clock          clock
	resendInterval time.Duration
	updateTicker   *time.Ticker
	// alertmanagers is a map of base URLs to Alertmanager clients
	alertmanagers    map[string]alertmanagerClientIFace
	alertmanagersMux sync.Mutex
	activeAlerts     map[alertCacheKey]*checkerAlert
	inactiveAlerts   map[alertCacheKey]*checkerAlert

	// updateLock guards update() to ensure we don't try to send multiple times concurrently.
	// Using an uint32/atomics, not a sync.Mutex, to ensure we can bail out if called manually while already running
	// TODO: once go 1.18 is released, replace with https://pkg.go.dev/sync@master#Mutex.TryLock
	updateLock uint32
	ctx        context.Context
	cancel     context.CancelFunc
	// wg is used by updateLoop, so that Stop can block until it finishes
	wg sync.WaitGroup

	baseLabels map[string]string
}

func NewAlertGenerator(store storage.Store, resendInterval time.Duration, alertmanagers []string,
	baseLabels map[string]string,
) *AlertGenerator {
	c := &AlertGenerator{
		clock:          new(wallClock),
		store:          store,
		resendInterval: resendInterval,
		activeAlerts:   make(map[alertCacheKey]*checkerAlert),
		inactiveAlerts: make(map[alertCacheKey]*checkerAlert),
		alertmanagers:  make(map[string]alertmanagerClientIFace),
		baseLabels:     baseLabels,
	}
	c.UpdateAlertmanagerURLs(alertmanagers)
	return c
}

func (a *AlertGenerator) UpdateAlertmanagerURLs(newURLs []string) {
	a.alertmanagersMux.Lock()
	defer a.alertmanagersMux.Unlock()

	present := make(map[string]struct{})
	for _, base := range newURLs {
		if _, ok := a.alertmanagers[base]; !ok {
			a.alertmanagers[base] = newAlertmanagerClient(base)
			present[base] = struct{}{}
		}
	}

	for base := range a.alertmanagers {
		if _, ok := present[base]; !ok {
			delete(a.alertmanagers, base)
		}
	}
}

// Start starts the periodic update loop for sending alerts to Alertmanager.
func (a *AlertGenerator) Start() {
	a.ctx, a.cancel = context.WithCancel(context.Background())
	go a.updateLoop()
}

// Stop stops periodic sending of alerts and blocks until the update loop has stopped.
func (a *AlertGenerator) Stop() {
	if a.ctx == nil {
		return
	}
	a.cancel()
	a.wg.Wait()
}

func (a *AlertGenerator) updateLoop() {
	zap.S().Infow("(Alertmanager) Starting update loop", "interval", a.resendInterval)
	a.wg.Add(1)
	defer a.wg.Done()
	a.updateTicker = time.NewTicker(a.resendInterval)
	defer a.updateTicker.Stop()
	for {
		select {
		case <-a.ctx.Done():
			zap.S().Warnw("(Alertmanager) Stopping update loop", "err", a.ctx.Err())
			return
		case <-a.updateTicker.C:
			zap.S().Debugw("(Alertmanager) Starting update")
			err := a.update(a.ctx)
			if err != nil {
				zap.S().Warnw("(Alertmanager) Update error", "err", err)
			} else {
				zap.S().Debugw("(Alertmanager) Update complete")
			}
		}
	}
}

// ManualUpdate pushes the current state of the checkers to Alertmanager outside the normal resend cycle.
// If an update is already in progress, ManualUpdate will return ErrAlreadyRunning.
func (a *AlertGenerator) ManualUpdate(ctx context.Context) error {
	zap.S().Infow("(Alertmanager) Updating manually")
	err := a.update(ctx)
	if err != nil {
		zap.S().Warnw("(Alertmanager) Manual update error", "err", err)
	} else {
		zap.S().Debugw("(Alertmanager) Manual update complete")
	}
	a.updateTicker.Reset(a.resendInterval)
	return err
}

// https://github.com/prometheus/compliance/blob/main/alert_generator/specification.md#conditions-for-sending-inactive-alerts
//
//nolint:lll
const maxInactiveAlertLifetime = 15 * time.Minute

var ErrAlreadyRunning = errors.New("already running")

func (a *AlertGenerator) update(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&a.updateLock, 0, 1) {
		return ErrAlreadyRunning
	}
	defer atomic.StoreUint32(&a.updateLock, 0)

	results, err := a.store.GetCheckerResult(values.CheckerSearch{})
	if err != nil {
		return fmt.Errorf("could not get checkers: %w", err)
	}

	var created, updated, inactivated, expired uint

	found := make(map[alertCacheKey]struct{})
	for _, result := range results {
		if result.Result.Status.Int() <= values.GoodCheckerStatus.Int() {
			continue
		}
		ca, err := newCheckerAlert(result, a.store)
		if err != nil {
			return fmt.Errorf("could not create checkerAlert (checker %v, cluster %v): %w", result.Result.Name,
				result.Cluster, err)
		}

		// Update it to ensure we pick up on any annotation changes
		// The cache key will ensure that a label change becomes a new alert rather than mutating the current one
		if _, ok := a.activeAlerts[ca.cacheKey()]; ok {
			updated++
		} else {
			created++
		}
		a.activeAlerts[ca.cacheKey()] = ca
		found[ca.cacheKey()] = struct{}{}
	}

	now := a.clock.Now()

	for key, alert := range a.activeAlerts {
		if _, ok := found[key]; ok {
			continue
		}
		alert.resolvedAt = &now
		a.inactiveAlerts[key] = alert
		delete(a.activeAlerts, key)
		inactivated++
	}

	for key, alert := range a.inactiveAlerts {
		if alert.resolvedAt.Before(now.Add(-1 * maxInactiveAlertLifetime)) {
			delete(a.inactiveAlerts, key)
			expired++
		}
	}

	zap.S().Debugw("(Alertmanager) Alert processing metrics",
		"created", created,
		"updated", updated,
		"inactivated", inactivated,
		"expired", expired)

	// Bail out if we don't need to send anything.
	// However, ensure that we send at least an empty array if we inactivated or expired any,
	// to prevent them from getting stuck
	if len(a.activeAlerts)+len(a.inactiveAlerts) == 0 && inactivated == 0 && expired == 0 {
		return nil
	}

	return a.sendAlertsToAlertmanager(ctx)
}

func (a *AlertGenerator) sendAlertsToAlertmanager(ctx context.Context) error {
	payload := make([]types.PostableAlert, 0, len(a.activeAlerts)+len(a.inactiveAlerts))
	for _, alert := range a.activeAlerts {
		payload = append(payload, alert.withBaseLabels(a.baseLabels).asPostableAlert())
	}
	for _, alert := range a.inactiveAlerts {
		payload = append(payload, alert.withBaseLabels(a.baseLabels).asPostableAlert())
	}

	a.alertmanagersMux.Lock()
	zap.S().Debugw("(Alertmanager) Sending alerts",
		"alerts", len(payload),
		"clients", len(a.alertmanagers))
	errsCh := make(chan error, len(a.alertmanagers))
	wg := sync.WaitGroup{}
	wg.Add(len(a.alertmanagers))
	for _, am := range a.alertmanagers {
		go func(am alertmanagerClientIFace) {
			defer wg.Done()
			err := am.PostAlerts(ctx, payload)
			if err != nil {
				errsCh <- fmt.Errorf("failed to send alerts to %s: %w", am.BaseURL(), err)
			}
		}(am)
	}
	a.alertmanagersMux.Unlock()

	wg.Wait()
	close(errsCh)

	if len(errsCh) == 0 {
		return nil
	}

	errs := make(multiError, 0, len(errsCh))
	for e := range errsCh {
		errs.add(e)
	}
	return errs
}
