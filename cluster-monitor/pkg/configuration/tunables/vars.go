// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

// Package tunables exposes configuration options that shouldn't be necessary in normal operation, so aren't command
// line flags, but may still be helpful to modify (e.g. for debugging or troubleshooting). These are all defined as
// environment variables to allow overriding at runtime, while using the default value in normal operation.
package tunables

import (
	"time"
)

const varsPrefix = "CB_MULTI_"

var (
	AgentPortPingTimeout              = duration("AP_PING_TIMEOUT", 5*time.Second)
	AgentPortActivateTimeout          = duration("AP_ACTIVATE_TIMEOUT", 30*time.Second)
	AgentPortMaxRetries               = integer("AP_MAX_RETRIES", 3)
	ClusterManagerAgentCheckerTimeout = duration("CM_AGENT_CHECKER_TIMEOUT", time.Minute)
)
