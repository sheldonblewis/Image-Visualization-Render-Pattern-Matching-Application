package service

import (
	"testing"
)

func TestParsePercentPatternSuccess(t *testing.T) {
	cp, err := parsePattern("gs://bucket/root/%exp%/%class%_%idx%.jpg", ModePercent)
	if err != nil {
		t.Fatalf("parsePattern returned error: %v", err)
	}
	if cp.Bucket != "bucket" {
		t.Fatalf("bucket mismatch: %s", cp.Bucket)
	}
	if got := len(cp.CaptureNames); got != 3 {
		t.Fatalf("expected 3 captures, got %d", got)
	}
	if cp.Matcher == nil {
		t.Fatal("expected matcher")
	}
	if cp.LiteralPrefix != "root/" {
		t.Fatalf("literal prefix mismatch: %q", cp.LiteralPrefix)
	}
}

func TestParsePercentPatternDuplicateCapture(t *testing.T) {
	_, err := parsePattern("gs://bucket/%foo%/%foo%.jpg", ModePercent)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePercentPatternInvalidToken(t *testing.T) {
	_, err := parsePattern("gs://bucket/%Foo-1%/x.jpg", ModePercent)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
