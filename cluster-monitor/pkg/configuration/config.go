// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package configuration

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
)

type LabelSelectors map[string]string

// Strings is an alias for string[] that implements ArrayMarshaler.
type Strings []string

func (s Strings) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, str := range s {
		enc.AppendString(str)
	}
	return nil
}

// Config stores general configurable values for the manager
type Config struct {
	// Information about the store
	SQLiteKey string
	SQLiteDB  string
	// TLS certificate information
	CertPath string
	KeyPath  string
	// REST server information
	HTTPPort  int
	HTTPSPort int
	// Logging
	LogLevel zapcore.Level

	LogCheckLifetime time.Duration

	EncryptKey []byte
	SignKey    []byte
	UUID       string

	UIRoot string

	MaxWorkers int

	DisableHTTP  bool
	DisableHTTPS bool

	// Auto-provisioned admin credentials
	AdminUser     string
	AdminPassword string

	// Control the API made available
	EnableAdminAPI    bool
	EnableClusterAPI  bool
	EnableExtendedAPI bool

	PrometheusBaseURL       string
	PrometheusLabelSelector LabelSelectors
	AlertmanagerBaseLabels  LabelSelectors
	CouchbaseUser           string
	CouchbasePassword       string

	// Alertmanager pushing
	AlertmanagerURLs        Strings
	AlertmanagerResendDelay time.Duration
}

func (c *Config) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddBool("EnableAdminAPI", c.EnableAdminAPI)
	enc.AddBool("EnableClusterAPI", c.EnableClusterAPI)
	enc.AddBool("EnableExtendedAPI", c.EnableExtendedAPI)
	enc.AddString("AdminUser", c.AdminUser)
	enc.AddString("CertPath", c.CertPath)
	enc.AddString("KeyPath", c.KeyPath)
	enc.AddString("SQLiteDB", c.SQLiteDB)
	enc.AddString("UIRoot", c.UIRoot)
	enc.AddInt("HTTPPort", c.HTTPPort)
	enc.AddInt("HTTPSPort", c.HTTPSPort)
	enc.AddInt("MaxWorkers", c.MaxWorkers)
	enc.AddString("PrometheusBaseURL", c.PrometheusBaseURL)
	enc.AddString("PrometheusLabelSelector", fmt.Sprint(c.PrometheusLabelSelector))
	enc.AddString("CouchbaseUser", c.CouchbaseUser)
	_ = enc.AddArray("AlertmanagerURLs", c.AlertmanagerURLs)
	enc.AddDuration("AlertmanagerResendDelay", c.AlertmanagerResendDelay)
	enc.AddString("AlertmanagerBaseLabels", fmt.Sprint(c.AlertmanagerBaseLabels))
	enc.AddDuration("LogCheckLifetime", c.LogCheckLifetime)

	// Do not log these as protected:
	// enc.AddString("", c.AdminPassword)
	// enc.AddString("", c.SQLiteKey)
	// enc.AddString("", c.CouchbasePassword)

	return nil
}
