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

func BatchMetricNames(metricNames []string) [][]string {
	if len(metricNames) == 0 {
		return [][]string{}
	}

	batchSize := 15
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
