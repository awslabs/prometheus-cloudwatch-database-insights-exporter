package filter

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockFilterable struct {
	Fields map[string]string
	Tags   map[string]string
}

func (mockFilterable MockFilterable) GetFilterableFields() map[string]string {
	return mockFilterable.Fields
}

func (mockFilterable MockFilterable) GetFilterableTags() map[string]string {
	return mockFilterable.Tags
}

func TestNewPatternFilter(t *testing.T) {
	includePatterns := Patterns{
		"name": []*regexp.Regexp{regexp.MustCompile("^test-")},
	}
	excludePatterns := Patterns{
		"name": []*regexp.Regexp{regexp.MustCompile("^old-")},
	}

	filter := NewPatternFilter(includePatterns, excludePatterns)

	assert.NotNil(t, filter)
	assert.Implements(t, (*Filter)(nil), filter)

	assert.True(t, filter.HasFilters())
}

func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name            string
		includePatterns Patterns
		excludePatterns Patterns
		obj             Filterable
		expected        bool
	}{
		{
			name:            "nil object returns false",
			includePatterns: nil,
			excludePatterns: nil,
			obj:             nil,
			expected:        false,
		},
		{
			name:            "no patterns includes everything",
			includePatterns: nil,
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
			},
			expected: true,
		},
		{
			name: "include pattern matches",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^test-")},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
			},
			expected: true,
		},
		{
			name: "include pattern does not match",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^prod-")},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
			},
			expected: false,
		},
		{
			name:            "exclude pattern matches",
			includePatterns: nil,
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^temp-")},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "temp-instance"},
			},
			expected: false,
		},
		{
			name:            "exclude pattern does not match",
			includePatterns: nil,
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^temp-")},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-instance"},
			},
			expected: true,
		},
		{
			name: "exclude takes precedence over include when both match",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^prod-")},
			},
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("-db")},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-db-1"},
			},
			expected: false,
		},
		{
			name: "multiple include patterns - first matches",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^test-"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-instance"},
			},
			expected: true,
		},
		{
			name: "multiple include patterns - second matches",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^test-"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
			},
			expected: true,
		},
		{
			name: "multiple fields - all must match (AND logic)",
			includePatterns: Patterns{
				"name":   []*regexp.Regexp{regexp.MustCompile("^prod-")},
				"engine": []*regexp.Regexp{regexp.MustCompile("postgres")},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "test-instance",
					"engine": "postgres",
				},
			},
			expected: false,
		},
		{
			name: "tag-based filtering matches",
			includePatterns: Patterns{
				"tag.Environment": []*regexp.Regexp{regexp.MustCompile("^prod")},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
				Tags:   map[string]string{"Environment": "production"},
			},
			expected: true,
		},
		{
			name: "tag-based filtering does not match",
			includePatterns: Patterns{
				"tag.Environment": []*regexp.Regexp{regexp.MustCompile("^prod")},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
				Tags:   map[string]string{"Environment": "development"},
			},
			expected: false,
		},
		{
			name: "field not found returns false for include",
			includePatterns: Patterns{
				"nonexistent": []*regexp.Regexp{regexp.MustCompile(".*")},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
			},
			expected: false,
		},
		{
			name:            "field not found returns true for exclude",
			includePatterns: nil,
			excludePatterns: Patterns{
				"nonexistent": []*regexp.Regexp{regexp.MustCompile(".*")},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-instance"},
			},
			expected: true,
		},
		{
			name: "multiple fields with multiple patterns each - all match",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^test-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
					regexp.MustCompile("mysql"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "prod-instance",
					"engine": "postgres",
				},
			},
			expected: true,
		},
		{
			name: "multiple fields with multiple patterns each - first field matches, second doesn't",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^test-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("oracle"),
					regexp.MustCompile("sqlserver"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "prod-instance",
					"engine": "postgres",
				},
			},
			expected: false,
		},
		{
			name: "multiple fields with multiple patterns - complex include and exclude",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
					regexp.MustCompile("mysql"),
					regexp.MustCompile("mariadb"),
				},
			},
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("-temp$"),
					regexp.MustCompile("-old$"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "prod-db-instance",
					"engine": "postgres",
				},
			},
			expected: true,
		},
		{
			name: "multiple fields with multiple patterns - exclude takes precedence",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
					regexp.MustCompile("mysql"),
				},
			},
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("-temp$"),
					regexp.MustCompile("-old$"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "prod-db-instance",
					"engine": "postgres",
				},
			},
			expected: false,
		},
		{
			name: "three fields with multiple patterns each - all match",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^test-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
					regexp.MustCompile("mysql"),
				},
				"identifier": []*regexp.Regexp{
					regexp.MustCompile(".*-db-.*"),
					regexp.MustCompile(".*-cache-.*"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{
					"name":       "prod-instance",
					"engine":     "postgres",
					"identifier": "prod-db-001",
				},
			},
			expected: true,
		},
		{
			name: "three fields with multiple patterns each - one field doesn't match",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^test-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
					regexp.MustCompile("mysql"),
				},
				"identifier": []*regexp.Regexp{
					regexp.MustCompile(".*-db-.*"),
					regexp.MustCompile(".*-cache-.*"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{
					"name":       "prod-instance",
					"engine":     "oracle",
					"identifier": "prod-db-001",
				},
			},
			expected: false,
		},
		{
			name: "multiple exclude fields with multiple patterns each",
			includePatterns: nil,
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^temp-"),
					regexp.MustCompile("^old-"),
					regexp.MustCompile("-deprecated$"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("oracle-7"),
					regexp.MustCompile("mysql-5.0"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "prod-instance",
					"engine": "postgres",
				},
			},
			expected: true,
		},
		{
			name: "multiple exclude fields with multiple patterns - one exclude matches",
			includePatterns: nil,
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^temp-"),
					regexp.MustCompile("^old-"),
					regexp.MustCompile("-deprecated$"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("oracle-7"),
					regexp.MustCompile("mysql-5.0"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "temp-instance",
					"engine": "postgres",
				},
			},
			expected: false,
		},
		{
			name: "tag-based filtering with multiple fields and patterns",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
				},
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^prod"),
					regexp.MustCompile("^staging"),
				},
				"tag.Team": []*regexp.Regexp{
					regexp.MustCompile("backend"),
					regexp.MustCompile("frontend"),
					regexp.MustCompile("platform"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-instance"},
				Tags: map[string]string{
					"Environment": "production",
					"Team":        "backend",
				},
			},
			expected: true,
		},
		{
			name: "tag-based filtering with multiple fields - one tag doesn't match",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
				},
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^prod"),
					regexp.MustCompile("^staging"),
				},
				"tag.Team": []*regexp.Regexp{
					regexp.MustCompile("backend"),
					regexp.MustCompile("frontend"),
				},
			},
			excludePatterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-instance"},
				Tags: map[string]string{
					"Environment": "production",
					"Team":        "platform",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPatternFilter(tt.includePatterns, tt.excludePatterns)
			result := filter.ShouldInclude(tt.obj)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasFilters(t *testing.T) {
	tests := []struct {
		name            string
		includePatterns Patterns
		excludePatterns Patterns
		expected        bool
	}{
		{
			name:            "no patterns has no filters",
			includePatterns: nil,
			excludePatterns: nil,
			expected:        false,
		},
		{
			name:            "empty patterns has no filters",
			includePatterns: Patterns{},
			excludePatterns: Patterns{},
			expected:        false,
		},
		{
			name: "include patterns has filters",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^test-")},
			},
			excludePatterns: nil,
			expected:        true,
		},
		{
			name:            "exclude patterns has filters",
			includePatterns: nil,
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^temp-")},
			},
			expected: true,
		},
		{
			name: "both patterns have filters",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^test-")},
			},
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{regexp.MustCompile("^temp-")},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPatternFilter(tt.includePatterns, tt.excludePatterns)
			result := filter.HasFilters()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMatchesPatterns(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		patterns []*regexp.Regexp
		expected bool
	}{
		{
			name:     "empty patterns returns false",
			value:    "test-value",
			patterns: []*regexp.Regexp{},
			expected: false,
		},
		{
			name:     "nil patterns returns false",
			value:    "test-value",
			patterns: nil,
			expected: false,
		},
		{
			name:  "single pattern matches",
			value: "test-instance",
			patterns: []*regexp.Regexp{
				regexp.MustCompile("^test-"),
			},
			expected: true,
		},
		{
			name:  "single pattern does not match",
			value: "prod-instance",
			patterns: []*regexp.Regexp{
				regexp.MustCompile("^test-"),
			},
			expected: false,
		},
		{
			name:  "multiple patterns - first matches",
			value: "test-instance",
			patterns: []*regexp.Regexp{
				regexp.MustCompile("^test-"),
				regexp.MustCompile("^prod-"),
			},
			expected: true,
		},
		{
			name:  "multiple patterns - second matches",
			value: "prod-instance",
			patterns: []*regexp.Regexp{
				regexp.MustCompile("^test-"),
				regexp.MustCompile("^prod-"),
			},
			expected: true,
		},
		{
			name:  "multiple patterns - none match",
			value: "dev-instance",
			patterns: []*regexp.Regexp{
				regexp.MustCompile("^test-"),
				regexp.MustCompile("^prod-"),
			},
			expected: false,
		},
		{
			name:  "complex regex pattern matches",
			value: "db-postgres-prod-us-west-2",
			patterns: []*regexp.Regexp{
				regexp.MustCompile(".*-postgres-.*"),
			},
			expected: true,
		},
		{
			name:  "complex regex pattern does not match",
			value: "db-mysql-prod-us-west-2",
			patterns: []*regexp.Regexp{
				regexp.MustCompile(".*-postgres-.*"),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPatterns(tt.value, tt.patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}
