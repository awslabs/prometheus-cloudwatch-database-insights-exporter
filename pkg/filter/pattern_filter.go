package filter

import (
	"regexp"
	"strings"
)

const (
	TagPrefix = "tag."
)

type Patterns map[string][]*regexp.Regexp

type PatternFilter struct {
	IncludePatterns Patterns
	ExcludePatterns Patterns
}

func NewPatternFilter(includePatterns, excludePatterns Patterns) Filter {
	return &PatternFilter{
		IncludePatterns: includePatterns,
		ExcludePatterns: excludePatterns,
	}
}

func (patternFilter *PatternFilter) ShouldInclude(obj Filterable) bool {
	if obj == nil {
		return false
	}

	// Exclude patterns: ANY field match should exclude (OR logic)
	if len(patternFilter.ExcludePatterns) > 0 {
		if patternFilter.matchesAnyField(obj, patternFilter.ExcludePatterns) {
			return false
		}
	}

	// Include patterns: ALL fields must match (AND logic)
	if len(patternFilter.IncludePatterns) > 0 {
		return patternFilter.matchesAllFields(obj, patternFilter.IncludePatterns)
	}

	return true
}

func (patternFilter *PatternFilter) HasFilters() bool {
	return len(patternFilter.IncludePatterns) > 0 || len(patternFilter.ExcludePatterns) > 0
}

// matchesAnyField returns true if ANY field matches its patterns (OR logic)
// Used for exclude patterns: any match should exclude the object
func (patternFilter *PatternFilter) matchesAnyField(obj Filterable, patterns Patterns) bool {
	fieldMap := obj.GetFilterableFields()
	tagMap := obj.GetFilterableTags()

	for filterKey, regexPatterns := range patterns {
		var fieldValue string
		var exists bool

		if fieldValue, exists = fieldMap[filterKey]; !exists {
			if strings.HasPrefix(filterKey, TagPrefix) {
				tagKey := filterKey[len(TagPrefix):]
				fieldValue, exists = tagMap[tagKey]
			}
		}

		// If field exists and matches any pattern, return true immediately
		if exists && matchesPatterns(fieldValue, regexPatterns) {
			return true
		}
	}

	return false
}

// matchesAllFields returns true only if ALL fields match their patterns (AND logic)
// Used for include patterns: all fields must match to include the object
func (patternFilter *PatternFilter) matchesAllFields(obj Filterable, patterns Patterns) bool {
	fieldMap := obj.GetFilterableFields()
	tagMap := obj.GetFilterableTags()

	for filterKey, regexPatterns := range patterns {
		var fieldValue string
		var exists bool

		if fieldValue, exists = fieldMap[filterKey]; !exists {
			if strings.HasPrefix(filterKey, TagPrefix) {
				tagKey := filterKey[len(TagPrefix):]
				fieldValue, exists = tagMap[tagKey]
			}
		}

		// If field doesn't exist or doesn't match any pattern, return false
		if !exists || !matchesPatterns(fieldValue, regexPatterns) {
			return false
		}
	}

	return true
}

func matchesPatterns(value string, regexPatterns []*regexp.Regexp) bool {
	for _, pattern := range regexPatterns {
		if pattern != nil && pattern.MatchString(value) {
			return true
		}
	}
	return false
}
