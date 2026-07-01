package sanitize

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

var policy *bluemonday.Policy

func init() {
	policy = bluemonday.UGCPolicy()
	policy.AllowStandardURLs()
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowElements("article", "section", "header", "footer", "main", "nav", "figure", "figcaption")
	policy.AllowAttrs("loading", "src", "alt", "width", "height").OnElements("img")
	policy.AllowAttrs("href").OnElements("a")
	policy.AllowAttrs("src", "allowfullscreen", "frameborder").OnElements("iframe")
	policy.RequireNoFollowOnLinks(false)
	policy.AllowDataAttributes()
}

func Sanitize(html string) string {
	cleaned := policy.Sanitize(html)
	cleaned = strings.TrimSpace(cleaned)
	return cleaned
}

func SanitizeText(text string) string {
	return policy.Sanitize(text)
}
