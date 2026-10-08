package store

import (
	"testing"
	"time"
)

func TestParseTime(t *testing.T) {
	// 实际扫描得到的 RFC3339Nano（modernc 驱动把时间串解析成 time.Time，
	// database/sql 再格式化为 RFC3339Nano 字符串）
	got := parseTime("2026-07-01T14:48:31.252863Z")
	want := time.Date(2026, 7, 1, 14, 48, 31, 252863000, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseTime(RFC3339Nano Z) = %v, want %v", got, want)
	}

	// 带时区偏移
	got = parseTime("2026-07-01T22:48:31.252863+08:00")
	want = time.Date(2026, 7, 1, 22, 48, 31, 252863000, time.FixedZone("", 8*3600))
	if !got.Equal(want) {
		t.Fatalf("parseTime(RFC3339Nano +08:00) = %v, want %v", got, want)
	}

	// 无小数秒
	got = parseTime("2026-07-01T14:48:31Z")
	want = time.Date(2026, 7, 1, 14, 48, 31, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseTime(RFC3339 no-frac) = %v, want %v", got, want)
	}

	// SQLite datetime('now') 兜底格式（UTC）
	got = parseTime("2026-07-01 14:48:31")
	want = time.Date(2026, 7, 1, 14, 48, 31, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("parseTime(datetime-now) = %v, want %v", got, want)
	}
}
