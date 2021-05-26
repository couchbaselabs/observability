package couchbase

import (
	"encoding/json"
	"fmt"
)

func (c *Client) GetAutoFailOverSettings() (*AutoFailoverSettings, error) {
	res, err := c.get(AutoFailOverSettings)
	if err != nil {
		return nil, fmt.Errorf("could not get auto failover settings: %w", err)
	}

	var settings AutoFailoverSettings
	if err = json.Unmarshal(res.Body, &settings); err != nil {
		return nil, fmt.Errorf("could not unmarshall the auto failover settings: %w", err)
	}

	return &settings, nil
}
