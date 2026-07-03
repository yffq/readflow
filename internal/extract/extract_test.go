package extract

import (
	"testing"
)

func TestExtractTitleTag(t *testing.T) {
	tests := []struct {
		html     string
		expected string
	}{
		{
			`<html><head><title>百度网络监控工具开源第四弹：evr — 构造 VXLAN 探测</title></head></html>`,
			"百度网络监控工具开源第四弹：evr — 构造 VXLAN 探测",
		},
		{
			`<title>Simple Title</title>`,
			"Simple Title",
		},
		{
			`<title attr="value">With Attributes</title>`,
			"With Attributes",
		},
		{
			`<html><head><title>
Multi-line
Title
</title></head></html>`,
			"Multi-line\nTitle",
		},
		{
			`<html><head></head><body>No title here</body></html>`,
			"",
		},
	}

	for _, tt := range tests {
		got := extractTitleTag(tt.html)
		if got != tt.expected {
			t.Errorf("extractTitleTag(%q) = %q, want %q", tt.html, got, tt.expected)
		}
	}
}

func TestStripImgAttrs(t *testing.T) {
	tests := []struct {
		html     string
		expected string
	}{
		{
			`<img src="a.jpg" width="800" height="600">`,
			`<img src="a.jpg">`,
		},
		{
			`<img width="800" height="600" src="a.jpg">`,
			`<img src="a.jpg">`,
		},
		{
			`<img src="a.jpg" Width="800" HEIGHT="600">`,
			`<img src="a.jpg">`,
		},
		{
			`<img src="a.jpg" class="photo" width="800" height="600" alt="pic">`,
			`<img src="a.jpg" class="photo" alt="pic">`,
		},
		{
			`<img src="a.jpg">`,
			`<img src="a.jpg">`,
		},
		{
			`<p>Some text</p>`,
			`<p>Some text</p>`,
		},
	}

	for _, tt := range tests {
		got := stripImgAttrs(tt.html)
		if got != tt.expected {
			t.Errorf("stripImgAttrs(%q) = %q, want %q", tt.html, got, tt.expected)
		}
	}
}
