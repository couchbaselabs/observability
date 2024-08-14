// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package core

type AgentState string

const (
	AgentNotStarted AgentState = "not_started"
	AgentWaiting    AgentState = "waiting"
	AgentReady      AgentState = "ready"
)
