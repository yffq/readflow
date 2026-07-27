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

func TestValidatePublicURL(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
	}{
		{name: "https domain", rawURL: "https://example.com/article", wantErr: false},
		{name: "http domain", rawURL: "http://example.com/article", wantErr: false},
		{name: "missing scheme", rawURL: "example.com/article", wantErr: true},
		{name: "ftp scheme", rawURL: "ftp://example.com/file", wantErr: true},
		{name: "localhost", rawURL: "http://localhost:8080", wantErr: true},
		{name: "localhost suffix", rawURL: "http://app.localhost", wantErr: true},
		{name: "local suffix", rawURL: "http://printer.local", wantErr: true},
		{name: "loopback ipv4", rawURL: "http://127.0.0.1:8080", wantErr: true},
		{name: "private ipv4", rawURL: "http://192.168.1.10", wantErr: true},
		{name: "metadata ipv4", rawURL: "http://169.254.169.254/latest/meta-data", wantErr: true},
		{name: "private ipv6", rawURL: "http://[fd00::1]/", wantErr: true},
		{name: "public ipv4", rawURL: "http://93.184.216.34", wantErr: false},
		{name: "userinfo", rawURL: "https://user:pass@example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePublicURL(tt.rawURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidatePublicURL(%q) error = %v, wantErr %v", tt.rawURL, err, tt.wantErr)
			}
		})
	}
}
