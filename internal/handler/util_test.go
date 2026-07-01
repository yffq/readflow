package handler

import (
	"testing"
	"time"
)

func TestIntValue(t *testing.T) {
	if intValue("", 5) != 5 {
		t.Fatal("default value not returned")
	}
	if intValue("10", 0) != 10 {
		t.Fatal("parsed value not returned")
	}
	if intValue("abc", 3) != 3 {
		t.Fatal("invalid value should return default")
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		html     string
		expected int
	}{
		{"<p>Hello World</p>", 2},
		{"<p>One two three four five</p>", 5},
		{"", 0},
		{"<p></p>", 0},
		{"<p>Hello</p><p>World</p>", 2},
	}

	for _, tt := range tests {
		got := countWords(tt.html)
		if got != tt.expected {
			t.Errorf("countWords(%q) = %d, want %d", tt.html, got, tt.expected)
		}
	}
}

func TestEstimateReadTime(t *testing.T) {
	if rt := estimateReadTime(0); rt != "< 1 min" {
		t.Errorf("0 words: expected '< 1 min', got %s", rt)
	}
	if rt := estimateReadTime(100); rt != "< 1 min" {
		t.Errorf("100 words: expected '< 1 min', got %s", rt)
	}
	if rt := estimateReadTime(500); rt != "2 min" {
		t.Errorf("500 words: expected '2 min', got %s", rt)
	}
	if rt := estimateReadTime(1000); rt != "4 min" {
		t.Errorf("1000 words: expected '4 min', got %s", rt)
	}
}

func TestFormatRelativeTime(t *testing.T) {
	if rt := formatRelativeTime(30 * time.Second); rt != "just now" {
		t.Errorf("30s: expected 'just now', got %s", rt)
	}
	if rt := formatRelativeTime(5 * time.Minute); rt == "" {
		t.Error("5min: should not be empty")
	}
	if rt := formatRelativeTime(3 * time.Hour); rt == "" {
		t.Error("3h: should not be empty")
	}
	if rt := formatRelativeTime(5 * 24 * time.Hour); rt == "" {
		t.Error("5d: should not be empty")
	}
}

func TestNewID(t *testing.T) {
	id1 := newID()
	id2 := newID()
	if id1 == id2 {
		t.Fatal("IDs should be unique")
	}
	if len(id1) != 32 {
		t.Fatalf("expected 32-char hex ID, got %d", len(id1))
	}
}
