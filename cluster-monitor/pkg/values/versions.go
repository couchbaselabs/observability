// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Versions map[string]Version

type VersionOS struct {
	Prefix     string `json:"prefix"`
	Deprecated bool   `json:"deprecated"`
}

type Version struct {
	Build string      `json:"version"`
	EOM   time.Time   `json:"-"`
	EOS   time.Time   `json:"-"`
	OS    []VersionOS `json:"OS"`
}

//go:embed versions.json
var verByte []byte

// GAVersions holds the parsed version.json information file
var GAVersions Versions

func init() {
	var err error
	GAVersions, err = getVersions()
	if err != nil {
		GAVersions = make(Versions)
		zap.S().Errorw("(Values) Could not load GA versions", "err", err)
	}
}

// GetVersions reads the versions file and returns the parsed output
func getVersions() (Versions, error) {
	versionList, err := parseVersions(verByte)
	if err != nil {
		return nil, fmt.Errorf("could not parse versions: %w", err)
	}

	return versionList, nil
}

// eomEosDate is a helper to unwrap the EOM/EOS data from the support-secret-sauce format (an array of ints/strings).
type eomEosDate time.Time

func (e *eomEosDate) UnmarshalJSON(bytes []byte) error {
	// Two possible cases: an array of ints, representing the [year, month]
	// or an array of ["TBD", ""]
	// (there's also the case where it isn't present at all, but we can ignore that here)
	var data []interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}
	switch data[0].(type) {
	case string:
		// leave it as nil
		return nil
	case float64:
		*e = eomEosDate(time.Date(int(data[0].(float64)), time.Month(data[1].(float64)), 1, 0, 0, 0, 0, time.UTC))
		return nil
	default:
		return fmt.Errorf("unexpected type %T for %#v", data[0], data[0])
	}
}

// UnmarshalJSON unmarshals Versions from the support-secret-sauce data format.
func (v *Version) UnmarshalJSON(data []byte) (err error) {
	var overlay struct {
		Build string      `json:"version"`
		OS    []VersionOS `json:"OS"`
		EOM   eomEosDate  `json:"eom,omitempty"`
		EOS   eomEosDate  `json:"eol"`
	}
	if err := json.Unmarshal(data, &overlay); err != nil {
		return err
	}

	// secret-sauce formats a version as 6_6_3_9808, while we want 6.6.3-9808
	versionNumber := strings.Replace(overlay.Build, "_", ".", 2)
	versionNumber = strings.Replace(versionNumber, "_", "-", 1)

	v.Build = versionNumber
	v.OS = overlay.OS
	if !time.Time(overlay.EOM).IsZero() {
		v.EOM = time.Time(overlay.EOM)
	}

	v.EOS = time.Time(overlay.EOS)
	return nil
}

// parseVersions parses the versions file
func parseVersions(byteval []byte) (Versions, error) {
	var overlay []Version

	err := json.Unmarshal(byteval, &overlay)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal version data %w", err)
	}

	versionList := make(map[string]Version)
	for _, ver := range overlay {
		versionList[ver.Build] = ver
	}

	return versionList, nil
}
