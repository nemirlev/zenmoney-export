// Code generated by mockery v2.43.1. DO NOT EDIT.

package driver

import (
	driver "github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	mock "github.com/stretchr/testify/mock"
)

// Rows is an autogenerated mock type for the Rows type
type Rows struct {
	mock.Mock
}

type Rows_Expecter struct {
	mock *mock.Mock
}

func (_m *Rows) EXPECT() *Rows_Expecter {
	return &Rows_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with given fields:
func (_m *Rows) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rows_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type Rows_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *Rows_Expecter) Close() *Rows_Close_Call {
	return &Rows_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *Rows_Close_Call) Run(run func()) *Rows_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Rows_Close_Call) Return(_a0 error) *Rows_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_Close_Call) RunAndReturn(run func() error) *Rows_Close_Call {
	_c.Call.Return(run)
	return _c
}

// ColumnTypes provides a mock function with given fields:
func (_m *Rows) ColumnTypes() []driver.ColumnType {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for ColumnTypes")
	}

	var r0 []driver.ColumnType
	if rf, ok := ret.Get(0).(func() []driver.ColumnType); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]driver.ColumnType)
		}
	}

	return r0
}

// Rows_ColumnTypes_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ColumnTypes'
type Rows_ColumnTypes_Call struct {
	*mock.Call
}

// ColumnTypes is a helper method to define mock.On call
func (_e *Rows_Expecter) ColumnTypes() *Rows_ColumnTypes_Call {
	return &Rows_ColumnTypes_Call{Call: _e.mock.On("ColumnTypes")}
}

func (_c *Rows_ColumnTypes_Call) Run(run func()) *Rows_ColumnTypes_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Rows_ColumnTypes_Call) Return(_a0 []driver.ColumnType) *Rows_ColumnTypes_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_ColumnTypes_Call) RunAndReturn(run func() []driver.ColumnType) *Rows_ColumnTypes_Call {
	_c.Call.Return(run)
	return _c
}

// Columns provides a mock function with given fields:
func (_m *Rows) Columns() []string {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Columns")
	}

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// Rows_Columns_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Columns'
type Rows_Columns_Call struct {
	*mock.Call
}

// Columns is a helper method to define mock.On call
func (_e *Rows_Expecter) Columns() *Rows_Columns_Call {
	return &Rows_Columns_Call{Call: _e.mock.On("Columns")}
}

func (_c *Rows_Columns_Call) Run(run func()) *Rows_Columns_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Rows_Columns_Call) Return(_a0 []string) *Rows_Columns_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_Columns_Call) RunAndReturn(run func() []string) *Rows_Columns_Call {
	_c.Call.Return(run)
	return _c
}

// Err provides a mock function with given fields:
func (_m *Rows) Err() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Err")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rows_Err_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Err'
type Rows_Err_Call struct {
	*mock.Call
}

// Err is a helper method to define mock.On call
func (_e *Rows_Expecter) Err() *Rows_Err_Call {
	return &Rows_Err_Call{Call: _e.mock.On("Err")}
}

func (_c *Rows_Err_Call) Run(run func()) *Rows_Err_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Rows_Err_Call) Return(_a0 error) *Rows_Err_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_Err_Call) RunAndReturn(run func() error) *Rows_Err_Call {
	_c.Call.Return(run)
	return _c
}

// Next provides a mock function with given fields:
func (_m *Rows) Next() bool {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Next")
	}

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Rows_Next_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Next'
type Rows_Next_Call struct {
	*mock.Call
}

// Next is a helper method to define mock.On call
func (_e *Rows_Expecter) Next() *Rows_Next_Call {
	return &Rows_Next_Call{Call: _e.mock.On("Next")}
}

func (_c *Rows_Next_Call) Run(run func()) *Rows_Next_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Rows_Next_Call) Return(_a0 bool) *Rows_Next_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_Next_Call) RunAndReturn(run func() bool) *Rows_Next_Call {
	_c.Call.Return(run)
	return _c
}

// Scan provides a mock function with given fields: dest
func (_m *Rows) Scan(dest ...interface{}) error {
	var _ca []interface{}
	_ca = append(_ca, dest...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Scan")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(...interface{}) error); ok {
		r0 = rf(dest...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rows_Scan_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Scan'
type Rows_Scan_Call struct {
	*mock.Call
}

// Scan is a helper method to define mock.On call
//   - dest ...interface{}
func (_e *Rows_Expecter) Scan(dest ...interface{}) *Rows_Scan_Call {
	return &Rows_Scan_Call{Call: _e.mock.On("Scan",
		append([]interface{}{}, dest...)...)}
}

func (_c *Rows_Scan_Call) Run(run func(dest ...interface{})) *Rows_Scan_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]interface{}, len(args)-0)
		for i, a := range args[0:] {
			if a != nil {
				variadicArgs[i] = a.(interface{})
			}
		}
		run(variadicArgs...)
	})
	return _c
}

func (_c *Rows_Scan_Call) Return(_a0 error) *Rows_Scan_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_Scan_Call) RunAndReturn(run func(...interface{}) error) *Rows_Scan_Call {
	_c.Call.Return(run)
	return _c
}

// ScanStruct provides a mock function with given fields: dest
func (_m *Rows) ScanStruct(dest interface{}) error {
	ret := _m.Called(dest)

	if len(ret) == 0 {
		panic("no return value specified for ScanStruct")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(interface{}) error); ok {
		r0 = rf(dest)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rows_ScanStruct_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'ScanStruct'
type Rows_ScanStruct_Call struct {
	*mock.Call
}

// ScanStruct is a helper method to define mock.On call
//   - dest interface{}
func (_e *Rows_Expecter) ScanStruct(dest interface{}) *Rows_ScanStruct_Call {
	return &Rows_ScanStruct_Call{Call: _e.mock.On("ScanStruct", dest)}
}

func (_c *Rows_ScanStruct_Call) Run(run func(dest interface{})) *Rows_ScanStruct_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(interface{}))
	})
	return _c
}

func (_c *Rows_ScanStruct_Call) Return(_a0 error) *Rows_ScanStruct_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_ScanStruct_Call) RunAndReturn(run func(interface{}) error) *Rows_ScanStruct_Call {
	_c.Call.Return(run)
	return _c
}

// Totals provides a mock function with given fields: dest
func (_m *Rows) Totals(dest ...interface{}) error {
	var _ca []interface{}
	_ca = append(_ca, dest...)
	ret := _m.Called(_ca...)

	if len(ret) == 0 {
		panic("no return value specified for Totals")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(...interface{}) error); ok {
		r0 = rf(dest...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Rows_Totals_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Totals'
type Rows_Totals_Call struct {
	*mock.Call
}

// Totals is a helper method to define mock.On call
//   - dest ...interface{}
func (_e *Rows_Expecter) Totals(dest ...interface{}) *Rows_Totals_Call {
	return &Rows_Totals_Call{Call: _e.mock.On("Totals",
		append([]interface{}{}, dest...)...)}
}

func (_c *Rows_Totals_Call) Run(run func(dest ...interface{})) *Rows_Totals_Call {
	_c.Call.Run(func(args mock.Arguments) {
		variadicArgs := make([]interface{}, len(args)-0)
		for i, a := range args[0:] {
			if a != nil {
				variadicArgs[i] = a.(interface{})
			}
		}
		run(variadicArgs...)
	})
	return _c
}

func (_c *Rows_Totals_Call) Return(_a0 error) *Rows_Totals_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Rows_Totals_Call) RunAndReturn(run func(...interface{}) error) *Rows_Totals_Call {
	_c.Call.Return(run)
	return _c
}

// NewRows creates a new instance of Rows. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewRows(t interface {
	mock.TestingT
	Cleanup(func())
}) *Rows {
	mock := &Rows{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
