package enrichment

import (
	"net/url"
	"strings"
)

// RefererClassifier classifies traffic sources from referer URLs.
type RefererClassifier struct {
	searchEngines []string
	socialMedia   []string
	aiPlatforms   []string
}

// NewRefererClassifier creates a new RefererClassifier with predefined domain lists.
func NewRefererClassifier() *RefererClassifier {
	return &RefererClassifier{
		searchEngines: []string{
			"google.com",
			"bing.com",
			"yahoo.com",
			"duckduckgo.com",
			"baidu.com",
			"yandex.ru",
			"ecosia.org",
		},
		socialMedia: []string{
			"facebook.com",
			"twitter.com",
			"x.com",
			"instagram.com",
			"linkedin.com",
			"pinterest.com",
			"reddit.com",
			"tiktok.com",
			"youtube.com",
			"threads.net",
			"mastodon.social",
		},
		aiPlatforms: []string{
			"chatgpt.com",
			"claude.ai",
			"gemini.google.com",
			"perplexity.ai",
			"copilot.microsoft.com",
		},
	}
}

// ClassifySource classifies the traffic source from a referer URL.
// Returns "Search", "Social", "AI", "Direct", or "Referral".
func (r *RefererClassifier) ClassifySource(refererStr string) string {
	// Empty or missing referer = direct visit
	if refererStr == "" {
		return "Direct"
	}

	parsed, err := url.Parse(refererStr)
	if err != nil {
		return "Direct"
	}

	// Extract and normalize hostname
	hostname := strings.ToLower(parsed.Hostname())
	hostname = strings.TrimPrefix(hostname, "www.")

	// Check against each category
	for _, domain := range r.searchEngines {
		if strings.Contains(hostname, domain) {
			return "Search"
		}
	}

	for _, domain := range r.socialMedia {
		if strings.Contains(hostname, domain) {
			return "Social"
		}
	}

	for _, domain := range r.aiPlatforms {
		if strings.Contains(hostname, domain) {
			return "AI"
		}
	}

	// Default to referral
	return "Referral"
}
