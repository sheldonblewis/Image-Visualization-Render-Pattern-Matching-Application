package service

import (
	"fmt"
	"regexp"
	"strings"
)

func parseRegexPattern(raw string) (*compiledPattern, error) {
	if !strings.HasPrefix(raw, "gs://") {
		return nil, fmt.Errorf("pattern must start with gs://")
	}

	trimmed := strings.TrimPrefix(raw, "gs://")
	slash := strings.IndexByte(trimmed, '/')
	if slash == -1 {
		return nil, fmt.Errorf("pattern must include bucket and object path")
	}

	bucket := trimmed[:slash]
	objectPattern := strings.TrimPrefix(trimmed[slash+1:], "/")
	if objectPattern == "" {
		return nil, fmt.Errorf("object pattern is required")
	}

	normPattern := normalizeRegexPattern(objectPattern)
	anchored := ensureAnchored(normPattern)
	matcher, err := regexp.Compile(anchored)
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}

	literalPrefix := regexLiteralPrefix(normPattern)

	return &compiledPattern{
		Raw:           raw,
		Mode:          ModeRegex,
		Bucket:        bucket,
		ObjectPattern: objectPattern,
		Matcher:       matcher,
		CaptureNames:  collectCaptureNames(matcher.SubexpNames()),
		SubexpNames:   matcher.SubexpNames(),
		LiteralPrefix: literalPrefix,
	}, nil
}

func normalizeRegexPattern(pattern string) string {
	var builder strings.Builder
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '(' && i+2 < len(pattern) && pattern[i+1] == '?' && pattern[i+2] == '<' {
			builder.WriteString("(?P<")
			i += 2
			continue
		}
		builder.WriteByte(pattern[i])
	}
	return builder.String()
}

func ensureAnchored(pattern string) string {
	trimmed := strings.TrimPrefix(pattern, "^")
	trimmed = strings.TrimSuffix(trimmed, "$")
	return "^" + trimmed + "$"
}

func regexLiteralPrefix(pattern string) string {
	var builder strings.Builder
	escaped := false
	trimmed := strings.TrimPrefix(pattern, "^")
	trimmed = strings.TrimSuffix(trimmed, "$")
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		if escaped {
			builder.WriteByte(ch)
			escaped = false
			continue
		}
		if ch == '\\' {
			escaped = true
			continue
		}
		if strings.ContainsRune(".*+?[](){}|$^", rune(ch)) {
			break
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func collectCaptureNames(subexp []string) []string {
	var names []string
	seen := map[string]struct{}{}
	for _, name := range subexp {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return names
}
