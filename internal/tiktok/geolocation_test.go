package tiktok

import (
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
