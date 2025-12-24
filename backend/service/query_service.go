package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/worldlabs/image-grid-viewer/backend/config"
	"github.com/worldlabs/image-grid-viewer/backend/storage"
)

type jobKind string

const (
	jobKindSegment jobKind = "segment"
	jobKindObjects jobKind = "objects"
)

type listJob struct {
	Kind         jobKind `json:"kind"`
	SegmentIndex int     `json:"segmentIndex"`
	Prefix       string  `json:"prefix"`
	PageToken    string  `json:"pageToken,omitempty"`
}

type cursorState struct {
	Pattern string     `json:"pattern"`
	Mode    Mode       `json:"mode"`
	Bucket  string     `json:"bucket"`
	Jobs    []listJob  `json:"jobs"`
	Stats   QueryStats `json:"stats"`
}

type QueryService struct {
	cfg     config.Config
	storage storage.Client
}

func NewQueryService(cfg config.Config, storage storage.Client) *QueryService {
	return &QueryService{
		cfg:     cfg,
		storage: storage,
	}
}

func (qs *QueryService) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	pattern := strings.TrimSpace(req.Pattern)
	if pattern == "" {
		return nil, newClientError("pattern is required")
	}

	mode, err := ParseMode(req.Mode)
	if err != nil {
		return nil, err
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = qs.cfg.DefaultPageSize
	}
	if pageSize < qs.cfg.MinPageSize {
		pageSize = qs.cfg.MinPageSize
	}
	if pageSize > qs.cfg.MaxPageSize {
		pageSize = qs.cfg.MaxPageSize
	}

	cp, err := parsePattern(pattern, mode)
	if err != nil {
		return nil, newClientError("%v", err)
	}

	var jobs []listJob
	stats := QueryStats{}
	if req.Cursor != "" {
		state, err := qs.decodeCursor(req.Cursor)
		if err != nil {
			return nil, newClientError("invalid cursor")
		}
		if state.Pattern != cp.Raw || state.Mode != cp.Mode || state.Bucket != cp.Bucket {
			return nil, newClientError("cursor does not match current pattern")
		}
		jobs = state.Jobs
		stats = state.Stats
	} else {
		jobs = qs.buildInitialJobs(cp)
	}

	items := make([]QueryItem, 0, pageSize)

	for len(items) < pageSize && len(jobs) > 0 {
		job := jobs[0]
		jobs = jobs[1:]

		switch job.Kind {
		case jobKindSegment:
			additionalJobs, err := qs.processSegmentJob(ctx, cp, job, &stats)
			if err != nil {
				return nil, err
			}
			jobs = append(jobs, additionalJobs...)
		case jobKindObjects:
			newItems, nextJobs, err := qs.processObjectsJob(ctx, cp, job, pageSize-len(items), &stats)
			if err != nil {
				return nil, err
			}
			items = append(items, newItems...)
			jobs = append(jobs, nextJobs...)
		default:
			return nil, fmt.Errorf("unknown job kind: %s", job.Kind)
		}
	}

	var nextCursor *string
	if len(jobs) > 0 {
		cursorValue, err := qs.encodeCursor(cursorState{
			Pattern: cp.Raw,
			Mode:    cp.Mode,
			Bucket:  cp.Bucket,
			Jobs:    jobs,
			Stats:   stats,
		})
		if err != nil {
			return nil, err
		}
		nextCursor = &cursorValue
	}

	return &QueryResponse{
		CaptureNames: cp.CaptureNames,
		Items:        items,
		NextCursor:   nextCursor,
		Stats:        stats,
	}, nil
}

func (qs *QueryService) buildInitialJobs(cp *compiledPattern) []listJob {
	if len(cp.Segments) == 0 {
		return []listJob{{
			Kind:         jobKindObjects,
			SegmentIndex: -1,
			Prefix:       strings.TrimSuffix(cp.LiteralPrefix, "/"),
		}}
	}

	var jobs []listJob
	prefix, idx := advanceLiteralSegments("", 0, cp.Segments)
	if idx >= len(cp.Segments) {
		return jobs
	}
	if idx == len(cp.Segments)-1 {
		jobs = append(jobs, listJob{
			Kind:         jobKindObjects,
			SegmentIndex: idx,
			Prefix:       prefix,
		})
	} else {
		jobs = append(jobs, listJob{
			Kind:         jobKindSegment,
			SegmentIndex: idx,
			Prefix:       prefix,
		})
	}
	return jobs
}

func (qs *QueryService) processSegmentJob(ctx context.Context, cp *compiledPattern, job listJob, stats *QueryStats) ([]listJob, error) {
	if job.SegmentIndex < 0 || job.SegmentIndex >= len(cp.Segments) {
		return nil, fmt.Errorf("segment index out of range")
	}
	seg := cp.Segments[job.SegmentIndex]
	basePrefix := job.Prefix
	listPrefix := joinPath(basePrefix, seg.LiteralPrefix)

	resp, err := qs.storage.List(ctx, storage.ListRequest{
		Bucket:    cp.Bucket,
		Prefix:    ensureTrailingSlash(listPrefix),
		Delimiter: "/",
		PageToken: job.PageToken,
		PageSize:  qs.cfg.MaxPageSize,
	})
	if err != nil {
		return nil, err
	}

	stats.ScannedPrefixes += len(resp.Prefixes)

	var newJobs []listJob
	for _, prefix := range resp.Prefixes {
		segmentValue := strings.TrimSuffix(strings.TrimPrefix(prefix, ensureTrailingSlash(basePrefix)), "/")
		if segmentValue == "" {
			continue
		}
		if !seg.Regex.MatchString(segmentValue) {
			continue
		}

		nextPrefix := joinPath(basePrefix, segmentValue)
		nextIndex := job.SegmentIndex + 1
		nextPrefix, nextIndex = advanceLiteralSegments(nextPrefix, nextIndex, cp.Segments)

		if nextIndex >= len(cp.Segments) {
			continue
		}

		if nextIndex == len(cp.Segments)-1 {
			newJobs = append(newJobs, listJob{
				Kind:         jobKindObjects,
				SegmentIndex: nextIndex,
				Prefix:       nextPrefix,
			})
		} else {
			newJobs = append(newJobs, listJob{
				Kind:         jobKindSegment,
				SegmentIndex: nextIndex,
				Prefix:       nextPrefix,
			})
		}
	}

	if resp.NextPageToken != "" {
		newJobs = append(newJobs, listJob{
			Kind:         job.Kind,
			SegmentIndex: job.SegmentIndex,
			Prefix:       job.Prefix,
			PageToken:    resp.NextPageToken,
		})
	}

	return newJobs, nil
}

func (qs *QueryService) processObjectsJob(ctx context.Context, cp *compiledPattern, job listJob, limit int, stats *QueryStats) ([]QueryItem, []listJob, error) {
	if limit <= 0 {
		return nil, []listJob{job}, nil
	}

	var objectPrefix string
	if len(cp.Segments) == 0 || job.SegmentIndex < 0 {
		objectPrefix = cp.LiteralPrefix
	} else {
		if job.SegmentIndex >= len(cp.Segments) {
			return nil, nil, fmt.Errorf("segment index out of range")
		}
		finalSeg := cp.Segments[job.SegmentIndex]
		basePrefix := job.Prefix
		objectPrefix = joinPath(basePrefix, finalSeg.LiteralPrefix)
	}

	resp, err := qs.storage.List(ctx, storage.ListRequest{
		Bucket:    cp.Bucket,
		Prefix:    objectPrefix,
		PageToken: job.PageToken,
		PageSize:  min(limit, qs.cfg.MaxPageSize),
	})
	if err != nil {
		return nil, nil, err
	}

	var items []QueryItem
	for _, obj := range resp.Objects {
		stats.ScannedObjects++
		matches := cp.Matcher.FindStringSubmatch(obj.Name)
		if matches == nil {
			continue
		}
		captures := make(map[string]string, len(cp.CaptureNames))
		for i, name := range cp.SubexpNames {
			if i == 0 || name == "" {
				continue
			}
			if i < len(matches) {
				captures[name] = matches[i]
			}
		}
		items = append(items, QueryItem{
			Object:   obj.Name,
			URL:      fmt.Sprintf("https://storage.googleapis.com/%s/%s", cp.Bucket, obj.Name),
			Captures: captures,
		})
		stats.Matched++
		if len(items) >= limit {
			break
		}
	}

	if resp.NextPageToken != "" {
		return items, []listJob{{
			Kind:         job.Kind,
			SegmentIndex: job.SegmentIndex,
			Prefix:       job.Prefix,
			PageToken:    resp.NextPageToken,
		}}, nil
	}

	return items, nil, nil
}

func (qs *QueryService) decodeCursor(encoded string) (*cursorState, error) {
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	var state cursorState
	if err := json.Unmarshal(bytes, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (qs *QueryService) encodeCursor(state cursorState) (string, error) {
	data, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func advanceLiteralSegments(prefix string, idx int, segments []segment) (string, int) {
	for idx < len(segments)-1 && !segments[idx].HasCapture {
		prefix = joinPath(prefix, segments[idx].Raw)
		idx++
	}
	return prefix, idx
}

func joinPath(base, part string) string {
	if base == "" {
		return part
	}
	if part == "" {
		return base
	}
	return base + "/" + part
}

func ensureTrailingSlash(value string) string {
	if value == "" {
		return value
	}
	if strings.HasSuffix(value, "/") {
		return value
	}
	return value + "/"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
