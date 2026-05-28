package idgen

import (
	"regexp"
	"testing"
	"time"
)

func TestNewUUIDv7Shape(t *testing.T) {
	got, err := NewUUIDv7At(time.UnixMilli(0x019b1fc92800))
	if err != nil {
		t.Fatalf("NewUUIDv7At() error = %v", err)
	}
	pattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !pattern.MatchString(got) {
		t.Fatalf("uuid = %q, want UUIDv7 shape", got)
	}
	if got[:8] != "019b1fc9" {
		t.Fatalf("uuid timestamp prefix = %q, want 019b1fc9", got[:8])
	}
}

func TestNewUUIDv7LaterTimeSortsAfterEarlierTime(t *testing.T) {
	earlier, err := NewUUIDv7At(time.UnixMilli(1000))
	if err != nil {
		t.Fatalf("NewUUIDv7At() error = %v", err)
	}
	later, err := NewUUIDv7At(time.UnixMilli(2000))
	if err != nil {
		t.Fatalf("NewUUIDv7At() error = %v", err)
	}
	if earlier >= later {
		t.Fatalf("earlier uuid %q should sort before later uuid %q", earlier, later)
	}
}
