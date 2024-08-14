// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func timeMust(t time.Time, err error) time.Time {
	if err != nil {
		panic(err)
	}
	return t
}

func stringsToOSes(versions []string) []VersionOS {
	result := make([]VersionOS, len(versions))
	for i, v := range versions {
		result[i] = VersionOS{Prefix: v}
	}
	return result
}

func TestParseEmbeddedVersions(t *testing.T) {
	versionList, err := parseVersions(verByte)
	require.NoError(t, err)
	for _, ver := range versionList {
		switch ver.Build {
		case "5.1.1-5905":
			t.Fatal("Found non-GA version in versions.json")
		case "6.6.3-9808":
			oses := stringsToOSes([]string{
				"Microsoft Windows Server 2016 Standard", "Microsoft Windows Server 2016 Datacenter",
				"Microsoft Windows Server 2019 Standard", "Microsoft Windows Server 2019 Datacenter",
				"Amazon Linux release 2", "CentOS Linux release 7.", "CentOS Linux release 8.",
				"Debian GNU/Linux 9.", "Debian GNU/Linux 10 ", "openSUSE 11.", "openSUSE 12.",
				"SUSE Linux Enterprise Server 12", "SUSE Linux Enterprise Server 15",
				"Oracle Linux Server release 7.", "Oracle Linux Server release 8.",
				"Red Hat Enterprise Linux Server release 7.", "Red Hat Enterprise Linux release 8.",
				"Ubuntu 16.", "Ubuntu 18.", "Ubuntu 20.",
			})
			for i, os := range oses {
				if os.Prefix == "Ubuntu 16." {
					oses[i].Deprecated = true
				}
			}
			require.Equal(t, Version{
				Build: "6.6.3-9808",
				EOM:   timeMust(time.Parse("2006-01-02", "2023-01-01")),
				EOS:   timeMust(time.Parse("2006-01-02", "2023-10-01")),
				OS:    oses,
			}, ver)
			return
		}
	}
	t.Fatal("could not find required test version")
}
