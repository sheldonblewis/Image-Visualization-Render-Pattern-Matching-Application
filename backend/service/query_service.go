package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

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

type jobTask struct {
	job   listJob
	limit int
}

type jobOutcome struct {
	items   []QueryItem
	newJobs []listJob
	stats   QueryStats
	err     error
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

	workerCount := qs.cfg.WorkerCount
	if workerCount < 1 {
		workerCount = 1
	}

	objectBatchSize := qs.objectBatchSize(pageSize)

	for len(items) < pageSize && len(jobs) > 0 {
		remaining := pageSize - len(items)
		if remaining <= 0 {
			break
		}

		batchSize := workerCount
		if len(jobs) < batchSize {
			batchSize = len(jobs)
		}

		finalBatch := make([]jobTask, 0, batchSize)
		for len(finalBatch) < batchSize && len(jobs) > 0 {
			job := jobs[0]
			jobs = jobs[1:]
			task := jobTask{job: job}
			if job.Kind == jobKindObjects {
				task.limit = objectBatchSize
			}
			finalBatch = append(finalBatch, task)
		}

		if len(finalBatch) == 0 {
			break
		}

		outcomes := make(chan jobOutcome, len(finalBatch))
		var wg sync.WaitGroup

		for _, task := range finalBatch {
			wg.Add(1)
			go func(task jobTask) {
				defer wg.Done()
				localStats := QueryStats{}
				switch task.job.Kind {
				case jobKindSegment:
					additionalJobs, err := qs.processSegmentJob(ctx, cp, task.job, &localStats)
					outcomes <- jobOutcome{
						newJobs: additionalJobs,
						stats:   localStats,
						err:     err,
					}
				case jobKindObjects:
					newItems, nextJobs, err := qs.processObjectsJob(ctx, cp, task.job, task.limit, &localStats, true)
					outcomes <- jobOutcome{
						items:   newItems,
						newJobs: nextJobs,
						stats:   localStats,
						err:     err,
					}
				default:
					outcomes <- jobOutcome{
						err: fmt.Errorf("unknown job kind: %s", task.job.Kind),
					}
				}
			}(task)
		}

		wg.Wait()
		close(outcomes)

		for outcome := range outcomes {
			if outcome.err != nil {
				return nil, outcome.err
			}

			stats.ScannedPrefixes += outcome.stats.ScannedPrefixes
			stats.ScannedObjects += outcome.stats.ScannedObjects
			stats.Matched += outcome.stats.Matched

			if len(outcome.items) > 0 {
				available := pageSize - len(items)
				if available > 0 {
					if len(outcome.items) > available {
						items = append(items, outcome.items[:available]...)
					} else {
						items = append(items, outcome.items...)
					}
				}
			}

			if len(outcome.newJobs) > 0 {
				jobs = append(jobs, outcome.newJobs...)
			}
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

func (qs *QueryService) Count(ctx context.Context, req QueryRequest) (*CountResponse, error) {
	pattern := strings.TrimSpace(req.Pattern)
	if pattern == "" {
		return nil, newClientError("pattern is required")
	}

	mode, err := ParseMode(req.Mode)
	if err != nil {
		return nil, err
	}

	cp, err := parsePattern(pattern, mode)
	if err != nil {
		return nil, newClientError("%v", err)
	}

	jobs := qs.buildInitialJobs(cp)
	stats := QueryStats{}

	workerCount := qs.cfg.WorkerCount
	if workerCount < 1 {
		workerCount = 1
	}

	objectBatchSize := qs.objectBatchSize(qs.cfg.MaxPageSize)

	for len(jobs) > 0 {
		batchSize := workerCount
		if len(jobs) < batchSize {
			batchSize = len(jobs)
		}

		finalBatch := make([]jobTask, 0, batchSize)
		for len(finalBatch) < batchSize && len(jobs) > 0 {
			job := jobs[0]
			jobs = jobs[1:]
			task := jobTask{job: job}
			if job.Kind == jobKindObjects {
				task.limit = objectBatchSize
			}
			finalBatch = append(finalBatch, task)
		}

		outcomes := make(chan jobOutcome, len(finalBatch))
		var wg sync.WaitGroup

		for _, task := range finalBatch {
			wg.Add(1)
			go func(task jobTask) {
				defer wg.Done()
				localStats := QueryStats{}
				switch task.job.Kind {
				case jobKindSegment:
					additionalJobs, err := qs.processSegmentJob(ctx, cp, task.job, &localStats)
					outcomes <- jobOutcome{
						newJobs: additionalJobs,
						stats:   localStats,
						err:     err,
					}
				case jobKindObjects:
					_, nextJobs, err := qs.processObjectsJob(ctx, cp, task.job, task.limit, &localStats, false)
					outcomes <- jobOutcome{
						newJobs: nextJobs,
						stats:   localStats,
						err:     err,
					}
				default:
					outcomes <- jobOutcome{err: fmt.Errorf("unknown job kind: %s", task.job.Kind)}
				}
			}(task)
		}

		wg.Wait()
		close(outcomes)

		for outcome := range outcomes {
			if outcome.err != nil {
				return nil, outcome.err
			}
			stats.ScannedPrefixes += outcome.stats.ScannedPrefixes
			stats.ScannedObjects += outcome.stats.ScannedObjects
			stats.Matched += outcome.stats.Matched
			if len(outcome.newJobs) > 0 {
				jobs = append(jobs, outcome.newJobs...)
			}
		}
	}

	return &CountResponse{
		Total: stats.Matched,
		Stats: stats,
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

func (qs *QueryService) processObjectsJob(ctx context.Context, cp *compiledPattern, job listJob, limit int, stats *QueryStats, collect bool) ([]QueryItem, []listJob, error) {
	if limit <= 0 {
		limit = qs.cfg.MaxPageSize
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

	remaining := limit
	nextToken := job.PageToken
	pagesRemaining := qs.prefetchPageCount()
	var items []QueryItem

	for remaining > 0 && pagesRemaining > 0 {
		pageSize := min(remaining, qs.cfg.MaxPageSize)
		resp, err := qs.storage.List(ctx, storage.ListRequest{
			Bucket:    cp.Bucket,
			Prefix:    objectPrefix,
			PageToken: nextToken,
			PageSize:  pageSize,
		})
		if err != nil {
			return nil, nil, err
		}

		for _, obj := range resp.Objects {
			stats.ScannedObjects++
			matches := cp.Matcher.FindStringSubmatch(obj.Name)
			if matches == nil {
				continue
			}

			stats.Matched++
			if collect {
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
			}

			remaining--
			if remaining <= 0 {
				break
			}
		}

		if resp.NextPageToken == "" {
			nextToken = ""
			break
		}

		nextToken = resp.NextPageToken
		pagesRemaining--
		if remaining <= 0 {
			break
		}
	}

	var nextJobs []listJob
	if nextToken != "" {
		nextJobs = append(nextJobs, listJob{
			Kind:         job.Kind,
			SegmentIndex: job.SegmentIndex,
			Prefix:       job.Prefix,
			PageToken:    nextToken,
		})
	}

	return items, nextJobs, nil
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

func (qs *QueryService) objectBatchSize(pageSize int) int {
	workers := qs.cfg.WorkerCount
	if workers < 1 {
		workers = 1
	}
	share := (pageSize + workers - 1) / workers
	if share < qs.cfg.MinPageSize {
		share = qs.cfg.MinPageSize
	}
	if share > qs.cfg.MaxPageSize {
		share = qs.cfg.MaxPageSize
	}
	return share * qs.prefetchPageCount()
}

func (qs *QueryService) prefetchPageCount() int {
	if qs.cfg.PrefetchPages < 1 {
		return 1
	}
	return qs.cfg.PrefetchPages
}
