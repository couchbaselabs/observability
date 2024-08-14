// Code generated by mockery v2.14.0. DO NOT EDIT.

package mocks

import (
	context "context"

	cbrest "github.com/couchbase/tools-common/cbrest"

	couchbase "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/couchbase"

	io "io"

	mock "github.com/stretchr/testify/mock"

	time "time"

	values "github.com/couchbaselabs/cbmultimanager/cluster-monitor/pkg/values"
)

// ClientIFace is an autogenerated mock type for the ClientIFace type
type ClientIFace struct {
	mock.Mock
}

// GetAutoFailOverSettings provides a mock function with given fields:
func (_m *ClientIFace) GetAutoFailOverSettings() (*values.AutoFailoverSettings, error) {
	ret := _m.Called()

	var r0 *values.AutoFailoverSettings
	if rf, ok := ret.Get(0).(func() *values.AutoFailoverSettings); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*values.AutoFailoverSettings)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBootstrap provides a mock function with given fields:
func (_m *ClientIFace) GetBootstrap() time.Time {
	ret := _m.Called()

	var r0 time.Time
	if rf, ok := ret.Get(0).(func() time.Time); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(time.Time)
	}

	return r0
}

// GetBucketStats provides a mock function with given fields: bucketName
func (_m *ClientIFace) GetBucketStats(bucketName string) (*values.BucketStat, error) {
	ret := _m.Called(bucketName)

	var r0 *values.BucketStat
	if rf, ok := ret.Get(0).(func(string) *values.BucketStat); ok {
		r0 = rf(bucketName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*values.BucketStat)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(bucketName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetBucketsSummary provides a mock function with given fields:
func (_m *ClientIFace) GetBucketsSummary() (values.BucketsSummary, error) {
	ret := _m.Called()

	var r0 values.BucketsSummary
	if rf, ok := ret.Get(0).(func() values.BucketsSummary); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(values.BucketsSummary)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetClusterInfo provides a mock function with given fields:
func (_m *ClientIFace) GetClusterInfo() *couchbase.PoolsMetadata {
	ret := _m.Called()

	var r0 *couchbase.PoolsMetadata
	if rf, ok := ret.Get(0).(func() *couchbase.PoolsMetadata); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*couchbase.PoolsMetadata)
		}
	}

	return r0
}

// GetDiagLog provides a mock function with given fields: ctx
func (_m *ClientIFace) GetDiagLog(ctx context.Context) (io.ReadCloser, error) {
	ret := _m.Called(ctx)

	var r0 io.ReadCloser
	if rf, ok := ret.Get(0).(func(context.Context) io.ReadCloser); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetFTSIndexStatus provides a mock function with given fields:
func (_m *ClientIFace) GetFTSIndexStatus() (values.FTSIndexStatus, error) {
	ret := _m.Called()

	var r0 values.FTSIndexStatus
	if rf, ok := ret.Get(0).(func() values.FTSIndexStatus); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(values.FTSIndexStatus)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetIndexStatus provides a mock function with given fields:
func (_m *ClientIFace) GetIndexStatus() ([]*values.IndexStatus, error) {
	ret := _m.Called()

	var r0 []*values.IndexStatus
	if rf, ok := ret.Get(0).(func() []*values.IndexStatus); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*values.IndexStatus)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetIndexStorageStats provides a mock function with given fields:
func (_m *ClientIFace) GetIndexStorageStats() ([]*values.IndexStatsStorage, error) {
	ret := _m.Called()

	var r0 []*values.IndexStatsStorage
	if rf, ok := ret.Get(0).(func() []*values.IndexStatsStorage); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*values.IndexStatsStorage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetMetric provides a mock function with given fields: start, end, metricName, step
func (_m *ClientIFace) GetMetric(start string, end string, metricName string, step string) (*couchbase.Metric, error) {
	ret := _m.Called(start, end, metricName, step)

	var r0 *couchbase.Metric
	if rf, ok := ret.Get(0).(func(string, string, string, string) *couchbase.Metric); ok {
		r0 = rf(start, end, metricName, step)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*couchbase.Metric)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, string, string) error); ok {
		r1 = rf(start, end, metricName, step)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNodeStorage provides a mock function with given fields:
func (_m *ClientIFace) GetNodeStorage() (*values.Storage, error) {
	ret := _m.Called()

	var r0 *values.Storage
	if rf, ok := ret.Get(0).(func() *values.Storage); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*values.Storage)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetNodesSummary provides a mock function with given fields:
func (_m *ClientIFace) GetNodesSummary() (values.NodesSummary, error) {
	ret := _m.Called()

	var r0 values.NodesSummary
	if rf, ok := ret.Get(0).(func() values.NodesSummary); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(values.NodesSummary)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPoolsBucket provides a mock function with given fields:
func (_m *ClientIFace) GetPoolsBucket() ([]values.Bucket, error) {
	ret := _m.Called()

	var r0 []values.Bucket
	if rf, ok := ret.Get(0).(func() []values.Bucket); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]values.Bucket)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSASLLogs provides a mock function with given fields: ctx, logName
func (_m *ClientIFace) GetSASLLogs(ctx context.Context, logName string) (io.ReadCloser, error) {
	ret := _m.Called(ctx, logName)

	var r0 io.ReadCloser
	if rf, ok := ret.Get(0).(func(context.Context, string) io.ReadCloser); ok {
		r0 = rf(ctx, logName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(io.ReadCloser)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, logName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetServerGroups provides a mock function with given fields:
func (_m *ClientIFace) GetServerGroups() ([]values.ServerGroup, error) {
	ret := _m.Called()

	var r0 []values.ServerGroup
	if rf, ok := ret.Get(0).(func() []values.ServerGroup); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]values.ServerGroup)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetUILogs provides a mock function with given fields:
func (_m *ClientIFace) GetUILogs() ([]values.UILogEntry, error) {
	ret := _m.Called()

	var r0 []values.UILogEntry
	if rf, ok := ret.Get(0).(func() []values.UILogEntry); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]values.UILogEntry)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// PingService provides a mock function with given fields: service
func (_m *ClientIFace) PingService(service cbrest.Service) error {
	ret := _m.Called(service)

	var r0 error
	if rf, ok := ret.Get(0).(func(cbrest.Service) error); ok {
		r0 = rf(service)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

type mockConstructorTestingTNewClientIFace interface {
	mock.TestingT
	Cleanup(func())
}

// NewClientIFace creates a new instance of ClientIFace. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewClientIFace(t mockConstructorTestingTNewClientIFace) *ClientIFace {
	mock := &ClientIFace{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
