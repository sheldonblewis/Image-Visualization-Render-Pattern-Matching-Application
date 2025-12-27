package service

import "fmt"

// Mode represents the supported pattern modes.
type Mode string

const (
	ModePercent Mode = "percent"
	ModeRegex   Mode = "regex"
)

func ParseMode(value string) (Mode, error) {
	switch Mode(value) {
	case ModePercent:
		return ModePercent, nil
	case ModeRegex:
		return ModeRegex, nil
	case "":
		return ModePercent, nil
	default:
		return "", fmt.Errorf("unsupported mode: %s", value)
	}
}

// QueryRequest represents the backend query input.
type QueryRequest struct {
	Pattern  string `json:"pattern"`
	Mode     string `json:"mode"`
	PageSize int    `json:"pageSize"`
	Cursor   string `json:"cursor"`
}

// QueryItem represents a single matched object.
type QueryItem struct {
	Object   string            `json:"object"`
	URL      string            `json:"url"`
	Captures map[string]string `json:"captures"`
}

// QueryStats exposes diagnostic information.
type QueryStats struct {
	ScannedPrefixes int `json:"scannedPrefixes"`
	ScannedObjects  int `json:"scannedObjects"`
	Matched         int `json:"matched"`
}

// QueryResponse is the handler response payload.
type QueryResponse struct {
	CaptureNames []string    `json:"captureNames"`
	Items        []QueryItem `json:"items"`
	NextCursor   *string     `json:"nextCursor,omitempty"`
	Stats        QueryStats  `json:"stats"`
}

// CountResponse returns total matches for a given pattern.
type CountResponse struct {
	Total int        `json:"total"`
	Stats QueryStats `json:"stats"`
}
