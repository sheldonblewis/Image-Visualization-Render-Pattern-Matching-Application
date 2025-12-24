package service

import "testing"

func TestParseRegexPatternSuccess(t *testing.T) {
	raw := `gs://bucket/images/(?<class>[0-9]{4})/img_(?<idx>[0-9]{2})\.jpg`

	cp, err := parsePattern(raw, ModeRegex)
	if err != nil {
		t.Fatalf("parsePattern returned error: %v", err)
	}

	if cp.Bucket != "bucket" {
		t.Fatalf("expected bucket bucket, got %s", cp.Bucket)
	}
	if cp.ObjectPattern == "" {
		t.Fatalf("expected object pattern to be retained")
	}
	if len(cp.CaptureNames) != 2 {
		t.Fatalf("expected 2 capture names, got %d", len(cp.CaptureNames))
	}
	if cp.LiteralPrefix != "images/" {
		t.Fatalf("literal prefix mismatch: %q", cp.LiteralPrefix)
	}
	if cp.Matcher == nil {
		t.Fatalf("expected compiled matcher")
	}
}

func TestParseRegexPatternMissingBucket(t *testing.T) {
	if _, err := parsePattern("gs://bucket-only", ModeRegex); err == nil {
		t.Fatal("expected error for missing object path")
	}
}
