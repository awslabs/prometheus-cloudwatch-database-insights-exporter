package utils

import (
	"regexp"
	"strings"
)

func SnakeCase(input string) string {
	result := strings.ReplaceAll(input, ".", "_")

	validChars := regexp.MustCompile(`[^a-zA-Z0-9_:]`)
	result = validChars.ReplaceAllString(result, "")

	result = strings.ToLower(result)

	return result
}

func BatchMetricNames(metricNames []string, batchSize int) [][]string {
	if len(metricNames) == 0 || batchSize <= 0 {
		return [][]string{}
	}

	batches := make([][]string, 0, (len(metricNames)+batchSize-1)/batchSize)

	for i := 0; i < len(metricNames); i += batchSize {
		end := i + batchSize
		if end > len(metricNames) {
			end = len(metricNames)
		}
		batches = append(batches, metricNames[i:end])
	}

	return batches
}

func isRegexPattern(metricName string) bool {
	regexPattern := regexp.MustCompile(`[*+?^${}()|[\\\]]`)
	return regexPattern.MatchString(metricName)
}

func compileRegexPatterns(patterns []string) ([]*regexp.Regexp, error) {
	var regexPatterns []*regexp.Regexp
	for _, pattern := range patterns {
		regex, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		regexPatterns = append(regexPatterns, regex)
	}
	return regexPatterns, nil
}
