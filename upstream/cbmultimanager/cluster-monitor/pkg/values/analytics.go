// Copyright (C) 2022 Couchbase, Inc.
//
// Use of this software is subject to the Couchbase Inc. License Agreement
// which may be found at https://www.couchbase.com/LA03012021.

package values

type AnalyticsNodeDiagnostics struct {
	Runtime AnalyticsRuntime `json:"runtime"`
}

type AnalyticsRuntime struct {
	SystemProperties AnalyticsSystemProperties `json:"systemProperties"`
}

type AnalyticsSystemProperties struct {
	JavaVendor  string `json:"java.vendor"`
	JavaVersion string `json:"java.version"`
}

type JavaVersion struct {
	Major string
	Minor int
}

type JavaVendor struct {
	VendorName       string
	SupportedVersion []JavaVersion
}

var SupportedVendorsCB6_5AndBelow = []*JavaVendor{
	{
		VendorName: "Oracle Corporation",
		SupportedVersion: []JavaVersion{
			{Major: "1.8.0", Minor: 181},
			{Major: "1.11.0"},
		},
	},
	{
		VendorName:       "Eclipse Foundation",
		SupportedVersion: []JavaVersion{{Major: "1.8.0"}},
	},
	{
		VendorName:       "AdoptOpenJDK",
		SupportedVersion: []JavaVersion{{Major: "1.8.0"}},
	},
}

var SupportedVendorsCB6_6AndAbove = []*JavaVendor{
	{
		VendorName: "Oracle Corporation",
		SupportedVersion: []JavaVersion{
			{Major: "1.11.0"},
		},
	},
	{
		VendorName:       "Eclipse Foundation",
		SupportedVersion: []JavaVersion{{Major: "11"}},
	},
	{
		VendorName:       "AdoptOpenJDK",
		SupportedVersion: []JavaVersion{{Major: "11"}},
	},
}

func FindSupportedVendor(vendors []*JavaVendor, vendor string) *JavaVendor {
	for _, supportedVendor := range vendors {
		if vendor == supportedVendor.VendorName {
			return supportedVendor
		}
	}
	return nil
}
