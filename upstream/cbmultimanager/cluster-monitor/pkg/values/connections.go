// Copyright (C) 2021 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type MemcachedConnection struct {
	AgentName   string          `json:"agent_name"`
	Connection  string          `json:"connection"`
	Socket      int             `json:"socket"`
	Protocol    string          `json:"protocol"`
	ParentPort  int             `json:"parent_port"`
	PeerName    string          `json:"peername,omitempty"`
	SockName    string          `json:"sockname,omitempty"`
	Internal    bool            `json:"internal"`
	User        json.RawMessage `json:"user"`
	SSL         json.RawMessage `json:"ssl,omitempty"`
	State       string          `json:"state"`
	TotalRecv   uint64          `json:"total_recv"`
	TotalSent   uint64          `json:"total_send"`
	BucketIndex int             `json:"bucket_index"`
}

type ConnectionData struct {
	MemcachedConnection

	// The SDK details are derived from the agent name and are populated by the SetSDKData method.
	SDKName    string   `json:"sdk_name"`
	SDKVersion string   `json:"sdk_version"`
	Source     *Address `json:"source,omitempty"`
	Server     *Address `json:"target,omitempty"`
	SSLEnabled bool     `json:"ssl_enabled"`
}

type Address struct {
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func (c *ConnectionData) UnmarshalJSON(data []byte) error {
	err := json.Unmarshal(data, &c.MemcachedConnection)
	if err != nil {
		return err
	}

	if c.Internal {
		return nil
	}

	c.SDKName, c.SDKVersion = getSDKInfoFromAgentName(c.AgentName)

	// The parsing of the peer names is best effort if it fails instead of returning Address structures we keep the
	// original peername and sockname
	if c.Source, err = parsePeerName(c.PeerName); err == nil {
		c.PeerName = ""
	}

	if c.Server, err = parsePeerName(c.SockName); err == nil {
		c.SockName = ""
	}

	// The parsing of the SSL field is best effort if it fails so be it.
	c.SSLEnabled, err = parseSSL(c.SSL)
	if err == nil {
		c.SSL = nil
	}

	return nil
}

type ServerConnections struct {
	InternalConnections uint64            `json:"internal_connections"`
	Connections         []*ConnectionData `json:"connections"`
}

func getSDKInfoFromAgentName(agent string) (string, string) {
	if strings.Contains(agent, "cbbackupmgr-") {
		return "backup", agent[strings.LastIndex(agent, "cbbackupmgr-")+len("cbbackupmgr-"):]
	}

	// most sdks follow the design sdkname/version so we will do a best effort to parse that
	sdkParts := strings.SplitN(agent, "/", 2)
	if len(sdkParts) != 2 {
		return agent, ""
	}

	return sdkParts[0], sdkParts[1]
}

// parsePeerName get the peername/sockname and attempts to parse it and return an Address. On later versions the
// peername is a string containing "{"ip": "10.10.10.10", "port": 8091}" but in others it is just the string
// "10.10.10.10:8091".
func parsePeerName(peer string) (*Address, error) {
	var address *Address

	if err := json.Unmarshal([]byte(peer), &address); err == nil {
		return address, nil
	}

	host, port, err := net.SplitHostPort(peer)
	if err != nil {
		return nil, fmt.Errorf("could not parse peer name '%s': %w", peer, err)
	}

	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid port '%s': %w", port, err)
	}

	return &Address{IP: host, Port: portNum}, nil
}

// parseSSL tries and unmarshal the ssl field for the connections data. The field can have one of two formats:
// 1. A boolean
// 2. A JSON object of the form {"enabled": true}
// If the given message is not either of those two this function returns an error.
func parseSSL(message json.RawMessage) (bool, error) {
	var enabled struct {
		Enabled bool `json:"enabled"`
	}

	if json.Unmarshal(message, &enabled.Enabled) == nil {
		return enabled.Enabled, nil
	}

	if json.Unmarshal(message, &enabled) == nil {
		return enabled.Enabled, nil
	}

	return false, fmt.Errorf("invalid ssl field '%s'", message)
}
