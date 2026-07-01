package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func intValue(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func countWords(html string) int {
	text := html
	for _, tag := range []string{"<p>", "</p>", "<li>", "</li>", "<br>", "<br/>"} {
		text = strings.ReplaceAll(text, tag, " ")
	}
	// simple tag strip
	inTag := false
	var clean strings.Builder
	for _, r := range text {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			clean.WriteRune(r)
		}
	}
	words := strings.Fields(clean.String())
	return len(words)
}

func estimateReadTime(words int) string {
	if words == 0 {
		return "< 1 min"
	}
	minutes := words / 250
	if minutes < 1 {
		return "< 1 min"
	}
	return fmt.Sprintf("%d min", minutes)
}

func formatRelativeTime(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	default:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	}
}
