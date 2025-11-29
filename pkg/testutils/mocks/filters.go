package mocks

import (
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/filter"
	"github.com/stretchr/testify/mock"
)

// MockFilter is a mock implementation of the Filter interface
type MockFilter struct {
	mock.Mock
}

func (mockFilter *MockFilter) ShouldInclude(obj filter.Filterable) bool {
	args := mockFilter.Called(obj)
	return args.Bool(0)
}

func (mockFilter *MockFilter) HasFilters() bool {
	args := mockFilter.Called()
	return args.Bool(0)
}

func NewMockFilter() *MockFilter {
	return &MockFilter{}
}
