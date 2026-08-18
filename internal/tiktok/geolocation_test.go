package tiktok

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseUsername(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"@kingston_vibes", "kingston_vibes"},
		{"kingston_vibes", "kingston_vibes"},
		{"https://www.tiktok.com/@876yardie", "876yardie"},
		{"https://www.tiktok.com/@876yardie?lang=en", "876yardie"},
		{"https://vm.tiktok.com/@dancehallking/", "dancehallking"},
	}
	for _, c := range cases {
		got := ParseUsername(c.input)
		if got != c.want {
			t.Errorf("ParseUsername(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestContainsAny(t *testing.T) {
	ok, matched := ContainsAny("wagwan massive, Kingston in di house", []string{"kingston", "portmore"})
	if !ok || matched != "kingston" {
		t.Errorf("expected kingston match, got ok=%v matched=%q", ok, matched)
	}
	ok, _ = ContainsAny("hello from London", []string{"paris", "berlin"})
	if ok {
		t.Error("unexpected match")
	}
}

func TestPatternScoring_Jamaica(t *testing.T) {
	gl := NewGeoLocator()
	result := &GeoResult{CountryScores: make(map[string]float64)}

	// Simulate Jamaican bio text
	bio := "876 Kingston born, Dancehall queen, wagwan massive!! #jamaica #reggae"
	gl.scoreText(bio, "bio", 2.0, result)

	if result.CountryScores["Jamaica"] == 0 {
		t.Error("expected Jamaica to score from Jamaican bio, got 0")
	}
	// Jamaica should dominate
	jScore := result.CountryScores["Jamaica"]
	for country, score := range result.CountryScores {
		if country != "Jamaica" && score > jScore {
			t.Errorf("country %q scored %.2f > Jamaica %.2f", country, score, jScore)
		}
	}
}

func TestPatternScoring_PhoneCode(t *testing.T) {
	gl := NewGeoLocator()
	result := &GeoResult{CountryScores: make(map[string]float64)}
	gl.scoreText("call me at +1876 555 1234 anytime", "bio", 2.0, result)
	if result.CountryScores["Jamaica"] == 0 {
		t.Error("Jamaica should score for +1876 phone code")
	}
}

func TestExtractGmails(t *testing.T) {
	text := `contact me at yardie876@gmail.com or check kingstoncrew@hotmail.com`
	emails := ExtractGmails(text)
	if len(emails) != 1 || emails[0] != "yardie876@gmail.com" {
		t.Errorf("expected [yardie876@gmail.com], got %v", emails)
	}
}

func TestExtractHashtags(t *testing.T) {
	text := "#Jamaica #dancehall #876life some text #reggae"
	tags := ExtractHashtags(text)
	for _, want := range []string{"#jamaica", "#dancehall", "#876life", "#reggae"} {
		found := false
		for _, tag := range tags {
			if tag == want {
				found = true
			}
		}
		if !found {
			t.Errorf("expected tag %q in %v", want, tags)
		}
	}
}

func TestFinalise_HighConfidence(t *testing.T) {
	result := &GeoResult{
		CountryScores: map[string]float64{
			"Jamaica":    80.0,
			"Trinidad":   5.0,
			"UK":         3.0,
		},
	}
	gl := NewGeoLocator()
	gl.finalise(result)
	if result.BestGuess != "Jamaica" {
		t.Errorf("expected Jamaica, got %q", result.BestGuess)
	}
	if result.Confidence != "High" {
		t.Errorf("expected High confidence, got %q", result.Confidence)
	}
}

func TestStripHTML(t *testing.T) {
	input := `<p class="foo">Hello <b>World</b></p>`
	got := stripHTML(input)
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Errorf("stripHTML left HTML tags: %q", got)
	}
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "World") {
		t.Errorf("stripHTML removed text content: %q", got)
	}
}

func TestParseCDNToken_USEast(t *testing.T) {
	cdnURL := "https://v16m-webapp.tiktok.com/video/tos/useast2a/tos-useast2a-ve-0068c001/oQIFFE9N/"
	token, region := parseCDNToken(cdnURL)
	if token != "useast2a" {
		t.Errorf("expected token 'useast2a', got %q", token)
	}
	if !strings.Contains(region, "US East") {
		t.Errorf("expected US East region, got %q", region)
	}
}

func TestParseCDNToken_Maliva(t *testing.T) {
	cdnURL := "https://v19-webapp.tiktok.com/video/tos/maliva/tos-maliva-ve-0068c799/abc123/"
	token, region := parseCDNToken(cdnURL)
	if token != "maliva" {
		t.Errorf("expected token 'maliva', got %q (region %q)", token, region)
	}
}

func TestParseCDNToken_None(t *testing.T) {
	cdnURL := "https://example.com/video/unknown/path/"
	token, _ := parseCDNToken(cdnURL)
	if token != "" {
		t.Errorf("expected no token for unknown URL, got %q", token)
	}
}

func TestCountryFromCDNToken(t *testing.T) {
	cases := map[string]string{
		"useast2a":     "United States",
		"eu-west-2":    "United Kingdom",
		"maliva":       "Malaysia",
		"alisg":        "Singapore",
		"br":           "Brazil",
	}
	for token, want := range cases {
		got := CountryFromCDNToken(token)
		if got != want {
			t.Errorf("CountryFromCDNToken(%q) = %q, want %q", token, got, want)
		}
	}
}

func TestISOToCountry(t *testing.T) {
	if got := isoToCountry("JM"); got != "Jamaica" {
		t.Errorf("expected Jamaica, got %q", got)
	}
	if got := isoToCountry("GB"); got != "United Kingdom" {
		t.Errorf("expected United Kingdom, got %q", got)
	}
	if got := isoToCountry("XX"); got != "" {
		t.Errorf("expected empty for unknown code, got %q", got)
	}
}

// ─── Timezone tests ──────────────────────────────────────────────────────────

func TestAnalyzePostingTimes_Jamaica(t *testing.T) {
	// Create fake videos posted at 22:00 UTC-5 = 03:00 UTC
	// i.e. UTC hours 02, 03, 04 — these should fit UTC-5 best
	// (22:00 local = bedtime fringe, but let's push it into evening at UTC-4)
	// Use 22:00 local time with UTC-5 → 03:00 UTC
	baseUTC := int64(1700000000) // some fixed epoch
	hourUTC := baseUTC - (baseUTC % 86400) // start of that day UTC

	var videos []VideoItem
	// Post at 23:00 local Jamaica (UTC-5) = 04:00 UTC
	for i := 0; i < 10; i++ {
		videos = append(videos, VideoItem{
			VideoID:    fmt.Sprintf("v%d", i),
			CreateTime: hourUTC + 4*3600 + int64(i)*86400,
		})
	}
	// Also post at 20:00 local = 01:00 UTC
	for i := 0; i < 5; i++ {
		videos = append(videos, VideoItem{
			VideoID:    fmt.Sprintf("v2%d", i),
			CreateTime: hourUTC + 1*3600 + int64(i)*86400,
		})
	}

	pp := AnalyzePostingTimes(videos)
	if pp == nil {
		t.Fatal("expected PostingPattern, got nil")
	}
	if pp.TotalVideos != 15 {
		t.Errorf("expected 15 videos, got %d", pp.TotalVideos)
	}
	// Best offset should be UTC-5 (half = -10) or UTC-4 (half = -8)
	// which maps to Jamaica / Caribbean
	gotCountries := strings.Join(pp.LikelyCountries, " ")
	if !strings.Contains(gotCountries, "Jamaica") && !strings.Contains(gotCountries, "United States") {
		t.Errorf("expected Jamaica or US in likely countries, got: %v", pp.LikelyCountries)
	}
}

func TestAnalyzePostingTimes_TooFewVideos(t *testing.T) {
	videos := []VideoItem{
		{VideoID: "a", CreateTime: 1700000000},
		{VideoID: "b", CreateTime: 1700086400},
	}
	pp := AnalyzePostingTimes(videos)
	if pp != nil {
		t.Error("expected nil for < 3 videos")
	}
}

func TestHalfToUTCString(t *testing.T) {
	cases := map[int]string{
		-10: "UTC-5",
		-8:  "UTC-4",
		0:   "UTC+0",
		2:   "UTC+1",
		11:  "UTC+5:30",
	}
	for half, want := range cases {
		got := halfToUTCString(half)
		if got != want {
			t.Errorf("halfToUTCString(%d) = %q, want %q", half, got, want)
		}
	}
}

func TestActivityScore(t *testing.T) {
	// Evening should score highest
	if activityScore(20) <= activityScore(3) {
		t.Error("expected evening (20:00) to score higher than dead zone (03:00)")
	}
	// Dead zone should be negative
	if activityScore(4) >= 0 {
		t.Errorf("expected dead zone (04:00) to be negative, got %.1f", activityScore(4))
	}
}

// ─── Cross-platform tests ─────────────────────────────────────────────────────

func TestPlatformDomain(t *testing.T) {
	cases := map[string]string{
		"https://www.instagram.com/{}/": "instagram.com",
		"https://x.com/{}":              "x.com",
		"https://www.facebook.com/{}":   "facebook.com",
	}
	for urlFmt, want := range cases {
		got := platformDomain(urlFmt)
		if got != want {
			t.Errorf("platformDomain(%q) = %q, want %q", urlFmt, got, want)
		}
	}
}

func TestExtractFromPage_LocationMeta(t *testing.T) {
	html := `<html><head>
	<meta name="geo.region" content="JM-01" />
	<meta name="description" content="Kingston reggae artist" />
	</head><body></body></html>`
	bio, location, _ := extractFromPage(html, []string{`"description"`, `"location"`})
	if bio == "" && location == "" {
		t.Error("expected non-empty bio or location")
	}
}

func TestNormaliseVideo_POI(t *testing.T) {
	raw := apiVideoItem{
		ID:   "123456",
		Desc: "Kingston vibes #jamaica",
		POI: &struct {
			Name      string  `json:"name"`
			Address   string  `json:"address"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{
			Name:      "Kingston",
			Address:   "Kingston, Jamaica",
			Latitude:  17.9714,
			Longitude: -76.793,
		},
	}
	v := normaliseVideo(raw)
	if v.VideoID != "123456" {
		t.Errorf("unexpected VideoID: %q", v.VideoID)
	}
	if v.POI == nil {
		t.Fatal("expected POI, got nil")
	}
	if v.POI.Name != "Kingston" {
		t.Errorf("expected Kingston, got %q", v.POI.Name)
	}
}

