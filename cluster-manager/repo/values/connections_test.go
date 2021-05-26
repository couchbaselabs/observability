package values

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalConnection(t *testing.T) {
	type testCase struct {
		name        string
		dataIn      []byte
		expectError bool
		expected    *ConnectionData
	}

	cases := []testCase{
		{
			name: "ok-7.0",
			dataIn: []byte(`{"agent_name":"cbmultimanager/v0.0.0","connection":"alpha","socket":33,"internal":false,
							"ssl":true,"state":"running","user":{"domain":"local","user":"@ns_server"},
							"peername":"{\"port\":9000,\"ip\":\"127.0.0.1\"}",
							"sockname":"{\"ip\":\"::1\",\"port\":12000}"}`),
			expected: &ConnectionData{
				MemcachedConnection: MemcachedConnection{
					AgentName:  "cbmultimanager/v0.0.0",
					Connection: "alpha",
					Socket:     33,
					State:      "running",
					User:       []byte(`{"domain":"local","user":"@ns_server"}`),
				},
				SDKName:    "cbmultimanager",
				SDKVersion: "v0.0.0",
				Source:     &Address{IP: "127.0.0.1", Port: 9000},
				Server:     &Address{IP: "::1", Port: 12000},
				SSLEnabled: true,
			},
		},
		{
			name: "ok-6.0",
			dataIn: []byte(`{"agent_name":"cbmultimanager/v0.0.0","connection":"alpha","socket":33,"internal":false,
							"ssl":{"enabled": true},"state":"running","user":{"domain":"local","user":"@ns_server"},
							"peername":"127.0.0.1:9000","sockname":"[::1]:12000"}`),
			expected: &ConnectionData{
				MemcachedConnection: MemcachedConnection{
					AgentName:  "cbmultimanager/v0.0.0",
					Connection: "alpha",
					Socket:     33,
					State:      "running",
					User:       []byte(`{"domain":"local","user":"@ns_server"}`),
				},
				SDKName:    "cbmultimanager",
				SDKVersion: "v0.0.0",
				Source:     &Address{IP: "127.0.0.1", Port: 9000},
				Server:     &Address{IP: "::1", Port: 12000},
				SSLEnabled: true,
			},
		},
		{
			name: "ok-internal",
			dataIn: []byte(`{"agent_name":"cbmultimanager/v0.0.0","connection":"alpha","socket":33,"internal":true,
							"ssl":false,"state":"running","user":{"domain":"local","user":"@ns_server"},
							"peername":"127.0.0.1:9000","sockname":"[::1]:12000"}`),
			expected: &ConnectionData{
				MemcachedConnection: MemcachedConnection{
					AgentName:  "cbmultimanager/v0.0.0",
					Connection: "alpha",
					Socket:     33,
					State:      "running",
					User:       []byte(`{"domain":"local","user":"@ns_server"}`),
					Internal:   true,
					PeerName:   "127.0.0.1:9000",
					SockName:   "[::1]:12000",
					SSL:        json.RawMessage("false"),
				},
			},
		},
		{
			name:        "invalid-json",
			dataIn:      []byte(`{"agent_name":"alpha}`),
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out *ConnectionData
			err := json.Unmarshal(tc.dataIn, &out)
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, out)
		})
	}
}
