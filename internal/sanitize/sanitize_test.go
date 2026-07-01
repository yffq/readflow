package sanitize

import (
	"testing"
)

func TestSanitize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "basic paragraph",
			input:    "<p>Hello World</p>",
			contains: []string{"<p>Hello World</p>"},
		},
		{
			name:     "script tag removed",
			input:    "<p>Hello</p><script>alert('xss')</script>",
			contains: []string{"<p>Hello</p>"},
			excludes: []string{"<script", "alert"},
		},
		{
			name:     "event handler removed",
			input:    `<p onclick="alert('xss')">Hello</p>`,
			contains: []string{"<p>Hello</p>"},
			excludes: []string{"onclick"},
		},
		{
			name:     "safe links preserved",
			input:    `<a href="https://example.com">Example</a>`,
			contains: []string{"https://example.com"},
		},
		{
			name:     "javascript link removed",
			input:    `<a href="javascript:alert('xss')">click</a>`,
			excludes: []string{"javascript:"},
		},
		{
			name:     "images preserved",
			input:    `<img src="https://example.com/img.jpg" alt="test">`,
			contains: []string{"https://example.com/img.jpg", "alt=\"test\""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Sanitize(tt.input)
			for _, c := range tt.contains {
				if !containsStr(result, c) {
					t.Errorf("expected result to contain %q, got %q", c, result)
				}
			}
			for _, e := range tt.excludes {
				if containsStr(result, e) {
					t.Errorf("expected result to NOT contain %q, got %q", e, result)
				}
			}
		})
	}
}

func TestSanitizeText(t *testing.T) {
	result := SanitizeText("Hello <b>World</b>")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
}

func containsStr(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
