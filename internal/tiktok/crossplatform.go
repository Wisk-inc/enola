package tiktok

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// Platform describes one social network to probe.
type Platform struct {
	Name    string
	URLFmt  string   // profile URL template, {} = username
	BioKeys []string // HTML / JSON keys that contain bio / location text
}

// platforms is the list of social networks searched for cross-platform profiles.
var platforms = []Platform{
	{
		Name:    "Instagram",
		URLFmt:  "https://www.instagram.com/{}/",
		BioKeys: []string{`"biography"`, `"full_name"`, `"city_id"`, `"location_name"`, `"location"`},
	},
	{
		Name:    "Twitter / X",
		URLFmt:  "https://x.com/{}",
		BioKeys: []string{`"description"`, `"location"`, `"name"`, `"url"`},
	},
	{
		Name:    "YouTube",
		URLFmt:  "https://www.youtube.com/@{}",
		BioKeys: []string{`"description"`, `"country"`, `"keywords"`},
	},
	{
		Name:    "Facebook",
		URLFmt:  "https://www.facebook.com/{}",
		BioKeys: []string{`"location"`, `"city"`, `"country"`},
	},
	{
		Name:    "Snapchat",
		URLFmt:  "https://www.snapchat.com/add/{}",
		BioKeys: []string{`"displayName"`, `"bio"`, `"location"`},
	},
	{
		Name:    "LinkedIn",
		URLFmt:  "https://www.linkedin.com/in/{}/",
		BioKeys: []string{`"geoLocationName"`, `"geoCountryName"`, `"headline"`, `"summary"`},
	},
	{
		Name:    "Pinterest",
		URLFmt:  "https://www.pinterest.com/{}/",
		BioKeys: []string{`"location"`, `"about"`, `"country"`},
	},
	{
		Name:    "SoundCloud",
		URLFmt:  "https://soundcloud.com/{}",
		BioKeys: []string{`"country"`, `"city"`, `"description"`},
	},
}

// CrossPlatformHit records what was found on another social network.
type CrossPlatformHit struct {
	Platform  string
	URL       string
	Bio       string
	Location  string // explicit location field value, if found
	RawSnippet string // raw text excerpt for scoring
	Found     bool
}

// CrossPlatformFinder searches the same username on multiple social networks.
type CrossPlatformFinder struct {
	scraper *Scraper
}

// NewCrossPlatformFinder creates a CrossPlatformFinder.
func NewCrossPlatformFinder(scraper *Scraper) *CrossPlatformFinder {
	return &CrossPlatformFinder{scraper: scraper}
}

// Search probes all configured platforms for the given username.
// It first tries a DuckDuckGo search to confirm the profile exists, then
// fetches the profile page directly to extract bio/location text.
func (c *CrossPlatformFinder) Search(ctx context.Context, username string) []CrossPlatformHit {
	var hits []CrossPlatformHit

	for _, p := range platforms {
		profileURL := strings.ReplaceAll(p.URLFmt, "{}", username)

		// Quick DDG confirmation search first
		q := fmt.Sprintf(`site:%s "%s"`, platformDomain(p.URLFmt), username)
		ddgBody, err := c.scraper.SearchDDG(ctx, q)
		ddgConfirmed := err == nil && strings.Contains(strings.ToLower(ddgBody), strings.ToLower(username))

		// Fetch the profile page
		body, err := c.scraper.FetchPage(ctx, profileURL)
		if err != nil && !ddgConfirmed {
			continue
		}

		hit := CrossPlatformHit{
			Platform: p.Name,
			URL:      profileURL,
		}

		if body != "" {
			hit.Found = true
			hit.Bio, hit.Location, hit.RawSnippet = extractFromPage(body, p.BioKeys)
		} else if ddgConfirmed {
			hit.Found = true
			hit.RawSnippet = extractSearchSnippets(ddgBody, 2)[0]
		}

		if hit.Found {
			hits = append(hits, hit)
		}
	}

	// Also do a broader "username + location" web search across all platforms
	broader := c.broadSearch(ctx, username)
	hits = append(hits, broader...)

	return hits
}

// broadSearch fires targeted multi-platform DuckDuckGo queries.
func (c *CrossPlatformFinder) broadSearch(ctx context.Context, username string) []CrossPlatformHit {
	queries := []string{
		fmt.Sprintf(`"%s" instagram OR twitter OR youtube location country`, username),
		fmt.Sprintf(`"%s" facebook bio location city`, username),
		fmt.Sprintf(`"%s" soundcloud OR spotify biography country`, username),
	}

	var hits []CrossPlatformHit
	for _, q := range queries {
		body, err := c.scraper.SearchDDG(ctx, q)
		if err != nil {
			continue
		}
		snippets := extractSearchSnippets(body, 5)
		for _, s := range snippets {
			if s != "" {
				hits = append(hits, CrossPlatformHit{
					Platform:   "Web search",
					URL:        "",
					RawSnippet: s,
					Found:      true,
				})
			}
		}
	}
	return hits
}

// ─── JSON / HTML value extractors ────────────────────────────────────────────

var (
	// Matches: "key":"value" or "key": "value"
	reJSONStr = regexp.MustCompile(`"([^"]+)"\s*:\s*"([^"]*)"`)
	// <meta name="description" content="...">
	reMeta       = regexp.MustCompile(`(?i)<meta[^>]+name=["']description["'][^>]+content=["']([^"']+)["']`)
	reMetaOG     = regexp.MustCompile(`(?i)<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']+)["']`)
	reMetaCountry = regexp.MustCompile(`(?i)<meta[^>]+name=["'](?:country|region|geo\.region)["'][^>]+content=["']([^"']+)["']`)
	// location: "City, Country" patterns
	reLocationStr = regexp.MustCompile(`(?i)(?:location|city|country|region|based in|from)\s*[:\-–]\s*([A-Za-z ,]{3,50})`)
)

// extractFromPage pulls bio, explicit location, and a raw text snippet.
func extractFromPage(body string, bioKeys []string) (bio, location, snippet string) {
	// Try JSON keys first
	for _, m := range reJSONStr.FindAllStringSubmatch(body, -1) {
		if len(m) < 3 {
			continue
		}
		key, val := `"`+m[1]+`"`, m[2]
		if val == "" || len(val) > 500 {
			continue
		}
		for _, bk := range bioKeys {
			if strings.Contains(key, strings.Trim(bk, `"`)) {
				if strings.Contains(strings.ToLower(bk), "location") ||
					strings.Contains(strings.ToLower(bk), "city") ||
					strings.Contains(strings.ToLower(bk), "country") {
					if location == "" {
						location = val
					}
				} else if bio == "" {
					bio = val
				}
			}
		}
	}

	// Meta description tags
	if bio == "" {
		if m := reMeta.FindStringSubmatch(body); len(m) > 1 {
			bio = stripHTML(m[1])
		}
	}
	if bio == "" {
		if m := reMetaOG.FindStringSubmatch(body); len(m) > 1 {
			bio = stripHTML(m[1])
		}
	}

	// Explicit country/geo meta tag
	if location == "" {
		if m := reMetaCountry.FindStringSubmatch(body); len(m) > 1 {
			location = m[1]
		}
	}

	// Location pattern in text
	if location == "" {
		if m := reLocationStr.FindStringSubmatch(body); len(m) > 1 {
			location = strings.TrimSpace(m[1])
		}
	}

	// Build snippet
	parts := []string{}
	if bio != "" {
		parts = append(parts, bio)
	}
	if location != "" {
		parts = append(parts, "Location: "+location)
	}
	snippet = strings.Join(parts, " | ")
	return
}

// platformDomain extracts the domain from a URL template for DDG site: search.
func platformDomain(urlFmt string) string {
	urlFmt = strings.ReplaceAll(urlFmt, "{}", "x")
	u := strings.TrimPrefix(urlFmt, "https://www.")
	u = strings.TrimPrefix(u, "https://")
	if idx := strings.Index(u, "/"); idx != -1 {
		return u[:idx]
	}
	return u
}
