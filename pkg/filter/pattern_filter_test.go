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
			name:            "multiple exclude fields with multiple patterns each",
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
			name:            "multiple exclude fields with multiple patterns - one exclude matches",
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

func TestNoTagFilterPatterns(t *testing.T) {
	// Test that all instances are included when no tag filters specified

	tests := []struct {
		name     string
		patterns Patterns
		obj      Filterable
		expected bool
	}{
		{
			name:     "no tag filters - instance with tags included",
			patterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "test-instance"},
				Tags:   map[string]string{"Environment": "production"},
			},
			expected: true,
		},
		{
			name:     "no tag filters - instance without tags included",
			patterns: nil,
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "test-instance"},
				Tags:   map[string]string{},
			},
			expected: true,
		},
		{
			name: "field filters only - instance with tags included if field matches",
			patterns: Patterns{
				"identifier": []*regexp.Regexp{regexp.MustCompile("^prod-")},
			},
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "prod-instance"},
				Tags:   map[string]string{"Environment": "production"},
			},
			expected: true,
		},
		{
			name: "field filters only - instance without tags included if field matches",
			patterns: Patterns{
				"identifier": []*regexp.Regexp{regexp.MustCompile("^prod-")},
			},
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "prod-instance"},
				Tags:   map[string]string{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPatternFilter(tt.patterns, nil)
			result := filter.ShouldInclude(tt.obj)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSingleFieldMultiplePatterns_ORLogic(t *testing.T) {
	// Test that multiple patterns for a single field use OR logic
	// (any pattern matching means the field matches)

	tests := []struct {
		name            string
		includePatterns Patterns
		obj             Filterable
		expected        bool
		description     string
	}{
		{
			name: "single field with 3 patterns - first pattern matches",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
					regexp.MustCompile("^dev-"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-database-01"},
			},
			expected:    true,
			description: "should include because first pattern matches",
		},
		{
			name: "single field with 3 patterns - second pattern matches",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
					regexp.MustCompile("^dev-"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "staging-database-01"},
			},
			expected:    true,
			description: "should include because second pattern matches",
		},
		{
			name: "single field with 3 patterns - third pattern matches",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
					regexp.MustCompile("^dev-"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "dev-database-01"},
			},
			expected:    true,
			description: "should include because third pattern matches",
		},
		{
			name: "single field with 3 patterns - none match",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
					regexp.MustCompile("^dev-"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-database-01"},
			},
			expected:    false,
			description: "should exclude because no pattern matches",
		},
		{
			name: "single field with 5 patterns - middle pattern matches",
			includePatterns: Patterns{
				"engine": []*regexp.Regexp{
					regexp.MustCompile("^postgres"),
					regexp.MustCompile("^mysql"),
					regexp.MustCompile("^mariadb"),
					regexp.MustCompile("^aurora"),
					regexp.MustCompile("^sqlserver"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"engine": "mariadb-10.5"},
			},
			expected:    true,
			description: "should include because middle pattern matches",
		},
		{
			name: "single field with complex patterns - partial match on second pattern",
			includePatterns: Patterns{
				"identifier": []*regexp.Regexp{
					regexp.MustCompile(".*-primary-.*"),
					regexp.MustCompile(".*-replica-.*"),
					regexp.MustCompile(".*-standby-.*"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "db-replica-us-east-1"},
			},
			expected:    true,
			description: "should include because second pattern matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPatternFilter(tt.includePatterns, nil)
			result := filter.ShouldInclude(tt.obj)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestSingleTagMultiplePatterns_ORLogic(t *testing.T) {
	// Test that multiple patterns for a single tag use OR logic
	// (any pattern matching means the tag matches)

	tests := []struct {
		name            string
		includePatterns Patterns
		obj             Filterable
		expected        bool
		description     string
	}{
		{
			name: "single tag with 3 patterns - first pattern matches",
			includePatterns: Patterns{
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^production"),
					regexp.MustCompile("^staging"),
					regexp.MustCompile("^development"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Environment": "production"},
			},
			expected:    true,
			description: "should include because first tag pattern matches",
		},
		{
			name: "single tag with 3 patterns - second pattern matches",
			includePatterns: Patterns{
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^production"),
					regexp.MustCompile("^staging"),
					regexp.MustCompile("^development"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Environment": "staging"},
			},
			expected:    true,
			description: "should include because second tag pattern matches",
		},
		{
			name: "single tag with 3 patterns - third pattern matches",
			includePatterns: Patterns{
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^production"),
					regexp.MustCompile("^staging"),
					regexp.MustCompile("^development"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Environment": "development"},
			},
			expected:    true,
			description: "should include because third tag pattern matches",
		},
		{
			name: "single tag with 3 patterns - none match",
			includePatterns: Patterns{
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^production"),
					regexp.MustCompile("^staging"),
					regexp.MustCompile("^development"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Environment": "testing"},
			},
			expected:    false,
			description: "should exclude because no tag pattern matches",
		},
		{
			name: "single tag with 4 patterns - last pattern matches",
			includePatterns: Patterns{
				"tag.Team": []*regexp.Regexp{
					regexp.MustCompile("backend"),
					regexp.MustCompile("frontend"),
					regexp.MustCompile("platform"),
					regexp.MustCompile("data"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Team": "data-engineering"},
			},
			expected:    true,
			description: "should include because last tag pattern matches",
		},
		{
			name: "single tag with wildcard patterns - middle pattern matches",
			includePatterns: Patterns{
				"tag.Owner": []*regexp.Regexp{
					regexp.MustCompile(".*@company\\.com$"),
					regexp.MustCompile(".*@partner\\.com$"),
					regexp.MustCompile(".*@vendor\\.com$"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Owner": "team@partner.com"},
			},
			expected:    true,
			description: "should include because middle tag pattern matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPatternFilter(tt.includePatterns, nil)
			result := filter.ShouldInclude(tt.obj)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestExcludeMultiplePatterns_ORLogic(t *testing.T) {
	// Test that multiple exclude patterns for a single field/tag use OR logic
	// (any pattern matching means the field/tag is excluded)

	tests := []struct {
		name            string
		excludePatterns Patterns
		obj             Filterable
		expected        bool
		description     string
	}{
		{
			name: "single field with 3 exclude patterns - first matches",
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^temp-"),
					regexp.MustCompile("^old-"),
					regexp.MustCompile("-deprecated$"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "temp-database"},
			},
			expected:    false,
			description: "should exclude because first exclude pattern matches",
		},
		{
			name: "single field with 3 exclude patterns - second matches",
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^temp-"),
					regexp.MustCompile("^old-"),
					regexp.MustCompile("-deprecated$"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "old-database"},
			},
			expected:    false,
			description: "should exclude because second exclude pattern matches",
		},
		{
			name: "single field with 3 exclude patterns - third matches",
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^temp-"),
					regexp.MustCompile("^old-"),
					regexp.MustCompile("-deprecated$"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "database-deprecated"},
			},
			expected:    false,
			description: "should exclude because third exclude pattern matches",
		},
		{
			name: "single field with 3 exclude patterns - none match",
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^temp-"),
					regexp.MustCompile("^old-"),
					regexp.MustCompile("-deprecated$"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-database"},
			},
			expected:    true,
			description: "should include because no exclude pattern matches",
		},
		{
			name: "single tag with 4 exclude patterns - any matches",
			excludePatterns: Patterns{
				"tag.Status": []*regexp.Regexp{
					regexp.MustCompile("^deprecated"),
					regexp.MustCompile("^decommissioned"),
					regexp.MustCompile("^archived"),
					regexp.MustCompile("^deleted"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Status": "archived"},
			},
			expected:    false,
			description: "should exclude because third exclude tag pattern matches",
		},
		{
			name: "single tag with 4 exclude patterns - none match",
			excludePatterns: Patterns{
				"tag.Status": []*regexp.Regexp{
					regexp.MustCompile("^deprecated"),
					regexp.MustCompile("^decommissioned"),
					regexp.MustCompile("^archived"),
					regexp.MustCompile("^deleted"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags:   map[string]string{"Status": "active"},
			},
			expected:    true,
			description: "should include because no exclude tag pattern matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPatternFilter(nil, tt.excludePatterns)
			result := filter.ShouldInclude(tt.obj)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestCombinedMultiplePatterns_ORWithinKeyANDCrossKeys(t *testing.T) {
	// Test that demonstrates OR logic within a single key
	// and AND logic across different keys

	tests := []struct {
		name            string
		includePatterns Patterns
		excludePatterns Patterns
		obj             Filterable
		expected        bool
		description     string
	}{
		{
			name: "two fields each with multiple patterns - both fields match via OR",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
					regexp.MustCompile("^uat-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
					regexp.MustCompile("mysql"),
					regexp.MustCompile("aurora"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "staging-db-01",
					"engine": "aurora-postgresql",
				},
			},
			expected:    true,
			description: "should include: name matches 2nd pattern (OR), engine matches 3rd pattern (OR), both fields match (AND)",
		},
		{
			name: "two fields each with multiple patterns - first field matches, second doesn't",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
					regexp.MustCompile("^uat-"),
				},
				"engine": []*regexp.Regexp{
					regexp.MustCompile("postgres"),
					regexp.MustCompile("mysql"),
					regexp.MustCompile("aurora"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{
					"name":   "prod-db-01",
					"engine": "sqlserver",
				},
			},
			expected:    false,
			description: "should exclude: name matches (OR), but engine doesn't match any pattern (OR failed), AND requires both",
		},
		{
			name: "field and tag each with multiple patterns - both match",
			includePatterns: Patterns{
				"identifier": []*regexp.Regexp{
					regexp.MustCompile(".*-primary$"),
					regexp.MustCompile(".*-replica$"),
					regexp.MustCompile(".*-standby$"),
				},
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^prod"),
					regexp.MustCompile("^staging"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "db-instance-replica"},
				Tags:   map[string]string{"Environment": "production"},
			},
			expected:    true,
			description: "should include: identifier matches 2nd pattern (OR), tag matches 1st pattern (OR), both match (AND)",
		},
		{
			name: "field and tag each with multiple patterns - field matches, tag doesn't",
			includePatterns: Patterns{
				"identifier": []*regexp.Regexp{
					regexp.MustCompile(".*-primary$"),
					regexp.MustCompile(".*-replica$"),
					regexp.MustCompile(".*-standby$"),
				},
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^prod"),
					regexp.MustCompile("^staging"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "db-instance-primary"},
				Tags:   map[string]string{"Environment": "development"},
			},
			expected:    false,
			description: "should exclude: identifier matches (OR), but tag doesn't match any pattern (OR failed), AND requires both",
		},
		{
			name: "include with multiple patterns and exclude with multiple patterns",
			includePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("^prod-"),
					regexp.MustCompile("^staging-"),
				},
			},
			excludePatterns: Patterns{
				"name": []*regexp.Regexp{
					regexp.MustCompile("-temp$"),
					regexp.MustCompile("-test$"),
					regexp.MustCompile("-backup$"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "prod-db-temp"},
			},
			expected:    false,
			description: "should exclude: name matches include (OR), but also matches exclude (OR), exclude takes precedence",
		},
		{
			name: "three fields with multiple patterns each - all match via different patterns",
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
				"identifier": []*regexp.Regexp{
					regexp.MustCompile(".*-db-.*"),
					regexp.MustCompile(".*-cache-.*"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{
					"name":       "staging-instance",
					"engine":     "mariadb-10.5",
					"identifier": "app-cache-001",
				},
			},
			expected:    true,
			description: "should include: all three fields match via OR within each key, AND across keys",
		},
		{
			name: "multiple tags with multiple patterns each - all match",
			includePatterns: Patterns{
				"tag.Environment": []*regexp.Regexp{
					regexp.MustCompile("^prod"),
					regexp.MustCompile("^staging"),
					regexp.MustCompile("^uat"),
				},
				"tag.Team": []*regexp.Regexp{
					regexp.MustCompile("backend"),
					regexp.MustCompile("frontend"),
					regexp.MustCompile("platform"),
					regexp.MustCompile("data"),
				},
				"tag.CostCenter": []*regexp.Regexp{
					regexp.MustCompile("^CC-100"),
					regexp.MustCompile("^CC-200"),
				},
			},
			obj: MockFilterable{
				Fields: map[string]string{"name": "test-db"},
				Tags: map[string]string{
					"Environment": "uat-env",
					"Team":        "data-team",
					"CostCenter":  "CC-200-engineering",
				},
			},
			expected:    true,
			description: "should include: all three tags match via OR within each tag key, AND across tag keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewPatternFilter(tt.includePatterns, tt.excludePatterns)
			result := filter.ShouldInclude(tt.obj)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestInvalidRegexPatterns(t *testing.T) {
	// Test that invalid regex patterns fail compilation
	invalidPatterns := []string{
		"[",           // Unclosed bracket
		"(?P<",        // Incomplete named group
		"*",           // Invalid quantifier
		"(?",          // Incomplete group
		"\\",          // Trailing backslash
		"[z-a]",       // Invalid range
		"(?P<name>.*", // Unclosed named group
	}

	for _, pattern := range invalidPatterns {
		t.Run("invalid_pattern_"+pattern, func(t *testing.T) {
			_, err := regexp.Compile(pattern)
			assert.Error(t, err, "Expected pattern %q to fail compilation", pattern)
		})
	}
}

func TestUntaggedInstanceHandling(t *testing.T) {
	// Test that instances without tags fail tag-based include filters
	includePatterns := Patterns{
		"tag.Environment": []*regexp.Regexp{regexp.MustCompile(".*")}, // Match anything
	}

	filter := NewPatternFilter(includePatterns, nil)

	tests := []struct {
		name     string
		obj      Filterable
		expected bool
	}{
		{
			name: "instance with empty tags should be excluded",
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "test-1"},
				Tags:   map[string]string{}, // Empty tags
			},
			expected: false,
		},
		{
			name: "instance with nil tags should be excluded",
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "test-2"},
				Tags:   nil, // Nil tags
			},
			expected: false,
		},
		{
			name: "instance with different tag should be excluded",
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "test-3"},
				Tags:   map[string]string{"DifferentTag": "value"},
			},
			expected: false,
		},
		{
			name: "instance with matching tag should be included",
			obj: MockFilterable{
				Fields: map[string]string{"identifier": "test-4"},
				Tags:   map[string]string{"Environment": "value"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filter.ShouldInclude(tt.obj)
			assert.Equal(t, tt.expected, result)
		})
	}
}
