package service

import (
	"fmt"
	"regexp"
	"strings"
)

var captureNameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

// compiledPattern represents a parsed and compiled object pattern.
type compiledPattern struct {
	Raw            string
	Mode           Mode
	Bucket         string
	ObjectPattern  string
	Segments       []segment
	CaptureNames   []string
	Matcher        *regexp.Regexp
	SubexpNames    []string
	LiteralPrefix  string
	LiteralPattern string
}

type segment struct {
	Raw           string
	Regex         *regexp.Regexp
	RegexBody     string
	CaptureNames  []string
	LiteralPrefix string
	HasCapture    bool
}

func parsePattern(raw string, mode Mode) (*compiledPattern, error) {
	switch mode {
	case ModeRegex:
		return parseRegexPattern(raw)
	case ModePercent:
		fallthrough
	default:
		return parsePercentPattern(raw)
	}
}

func parsePercentPattern(raw string) (*compiledPattern, error) {
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

	rawSegments := strings.Split(objectPattern, "/")
	segments := make([]segment, len(rawSegments))
	var captureNames []string
	for i, segRaw := range rawSegments {
		seg, segCaptures, err := parsePercentSegment(segRaw)
		if err != nil {
			return nil, fmt.Errorf("segment %d: %w", i, err)
		}
		segments[i] = seg
		for _, name := range segCaptures {
			for _, existing := range captureNames {
				if existing == name {
					return nil, fmt.Errorf("duplicate capture name: %s", name)
				}
			}
			captureNames = append(captureNames, name)
		}
	}

	fullPattern := buildFullRegex(segments)
	matcher, err := regexp.Compile(fullPattern)
	if err != nil {
		return nil, fmt.Errorf("compile full regex: %w", err)
	}

	literalPrefix := buildLiteralPrefix(segments)
	literalPattern := fullPattern

	return &compiledPattern{
		Raw:            raw,
		Mode:           ModePercent,
		Bucket:         bucket,
		ObjectPattern:  objectPattern,
		Segments:       segments,
		CaptureNames:   captureNames,
		Matcher:        matcher,
		SubexpNames:    matcher.SubexpNames(),
		LiteralPrefix:  literalPrefix,
		LiteralPattern: literalPattern,
	}, nil
}

func parsePercentSegment(raw string) (segment, []string, error) {
	var builder strings.Builder
	var literalPrefix strings.Builder
	captures := []string{}
	seenCapture := false

	i := 0
	for i < len(raw) {
		ch := raw[i]
		if ch == '%' {
			if i+1 < len(raw) && raw[i+1] == '%' {
				builder.WriteString("%")
				if !seenCapture {
					literalPrefix.WriteByte('%')
				}
				i += 2
				continue
			}
			end := strings.IndexByte(raw[i+1:], '%')
			if end == -1 {
				return segment{}, nil, fmt.Errorf("unterminated capture token")
			}
			name := raw[i+1 : i+1+end]
			if name == "" {
				return segment{}, nil, fmt.Errorf("empty capture name")
			}
			if !captureNameRegex.MatchString(name) {
				return segment{}, nil, fmt.Errorf("invalid capture name: %s", name)
			}
			builder.WriteString(fmt.Sprintf("(?P<%s>[^/]+)", name))
			captures = append(captures, name)
			seenCapture = true
			i += end + 2
			continue
		}
		builder.WriteString(regexp.QuoteMeta(string(ch)))
		if !seenCapture {
			literalPrefix.WriteByte(ch)
		}
		i++
	}

	regexBody := builder.String()
	segRegex, err := regexp.Compile("^" + regexBody + "$")
	if err != nil {
		return segment{}, nil, err
	}

	return segment{
		Raw:           raw,
		Regex:         segRegex,
		RegexBody:     regexBody,
		CaptureNames:  captures,
		LiteralPrefix: literalPrefix.String(),
		HasCapture:    len(captures) > 0,
	}, captures, nil
}

func buildFullRegex(segments []segment) string {
	var builder strings.Builder
	builder.WriteString("^")
	for i, seg := range segments {
		if i > 0 {
			builder.WriteString("/")
		}
		builder.WriteString(seg.RegexBody)
	}
	builder.WriteString("$")
	return builder.String()
}

func buildLiteralPrefix(segments []segment) string {
	var builder strings.Builder
	for i, seg := range segments {
		if i > 0 {
			builder.WriteString("/")
		}
		builder.WriteString(seg.LiteralPrefix)
		if seg.HasCapture {
			break
		}
	}
	return builder.String()
}
