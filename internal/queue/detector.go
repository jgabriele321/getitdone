package queue

import (
	"regexp"
	"strings"
)

var (
	// Regular expressions for format detection
	bulletPointRegex = regexp.MustCompile(`(?m)^[-•*]\s+.+$`)
	andSplitRegex    = regexp.MustCompile(`(?i)\s+and\s+`)
	dotSplitRegex    = regexp.MustCompile(`\.\s+[A-Z]`)
)

// DetectMessageFormat analyzes a message and determines its format type
func DetectMessageFormat(message string) FormatType {
	// Check for bullet points
	if bulletPointRegex.MatchString(message) {
		return FormatBulletList
	}

	// Check for narrative with multiple tasks
	if dotSplitRegex.MatchString(message) && strings.Contains(strings.ToLower(message), " will ") {
		return FormatNarrativeMulti
	}

	// Check for tasks split by "AND"
	if andSplitRegex.MatchString(message) {
		return FormatNarrativeMulti
	}

	// Default to single task
	return FormatSingleTask
}

// SplitMessage splits a message into individual task messages based on its format
func SplitMessage(message string, format FormatType) []string {
	switch format {
	case FormatBulletList:
		return splitBulletList(message)
	case FormatNarrativeMulti:
		return splitNarrative(message)
	default:
		return []string{message}
	}
}

// splitBulletList splits a bullet-pointed list into individual tasks
func splitBulletList(message string) []string {
	var tasks []string
	lines := strings.Split(message, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if bulletPointRegex.MatchString(line) {
			// Remove the bullet point and trim
			task := strings.TrimSpace(line[1:])
			if task != "" {
				tasks = append(tasks, task)
			}
		}
	}

	return tasks
}

// splitNarrative splits a narrative message into individual tasks
func splitNarrative(message string) []string {
	// First try splitting by "AND"
	if andSplitRegex.MatchString(message) {
		parts := andSplitRegex.Split(message, -1)
		var tasks []string
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				tasks = append(tasks, part)
			}
		}
		return tasks
	}

	// Then try splitting by sentences that start with a capital letter
	matches := dotSplitRegex.FindAllStringIndex(message, -1)
	if len(matches) > 0 {
		var tasks []string
		lastIndex := 0

		for _, match := range matches {
			// Add the sentence up to the period
			task := strings.TrimSpace(message[lastIndex : match[0]+1])
			if task != "" {
				tasks = append(tasks, task)
			}
			lastIndex = match[0] + 1
		}

		// Add the last sentence
		if lastIndex < len(message) {
			task := strings.TrimSpace(message[lastIndex:])
			if task != "" {
				tasks = append(tasks, task)
			}
		}

		return tasks
	}

	// If no splits found, return original message
	return []string{message}
}

// IsLikelyBatchMessage does a quick check if a message might contain multiple tasks
func IsLikelyBatchMessage(message string) bool {
	// Check for obvious indicators of multiple tasks
	if bulletPointRegex.MatchString(message) {
		return true
	}

	if andSplitRegex.MatchString(message) {
		return true
	}

	if dotSplitRegex.MatchString(message) && strings.Contains(strings.ToLower(message), " will ") {
		return true
	}

	// Count potential task indicators
	indicators := []string{
		"then",
		"after that",
		"next",
		"finally",
		"lastly",
		"first",
		"second",
		"third",
	}

	count := 0
	messageLower := strings.ToLower(message)
	for _, indicator := range indicators {
		if strings.Contains(messageLower, indicator) {
			count++
		}
	}

	return count >= 2
}
