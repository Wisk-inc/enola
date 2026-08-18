package tiktok

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// tzCountries maps common UTC offsets (in half-hours, so offset 0 = UTC+0,
// offset -10 = UTC-5, offset -8 = UTC-4, etc.) to the countries most likely
// to be in that timezone. Half-hour granularity covers places like India
// (UTC+5:30) and Newfoundland (UTC-3:30).
var tzCountries = map[int][]string{
	// Americas
	-24: {"United States", "Canada"},                           // UTC-12
	-22: {"United States"},                                     // UTC-11
	-20: {"United States"},                                     // UTC-10 (Hawaii)
	-18: {"United States", "Canada"},                           // UTC-9 (Alaska)
	-16: {"United States", "Canada", "Mexico"},                 // UTC-8 (Pacific)
	-14: {"United States", "Canada", "Mexico"},                 // UTC-7 (Mountain)
	-12: {"United States", "Canada", "Mexico", "Guatemala",
		"El Salvador", "Honduras", "Costa Rica"},               // UTC-6 (Central)
	-10: {"Jamaica", "United States", "Canada", "Cuba",
		"Haiti", "Bahamas", "Cayman Islands", "Panama",
		"Ecuador", "Colombia", "Peru"},                         // UTC-5 (Eastern / Jamaica)
	-8: {"Trinidad and Tobago", "Barbados", "Guyana",
		"Venezuela", "Bolivia", "Dominican Republic",
		"Puerto Rico", "Antigua", "Saint Lucia",
		"Grenada", "Dominica", "Saint Vincent",
		"United States (ET summer)", "Canada (AT)"},            // UTC-4 (Atlantic)
	-7: {"Canada"},                                             // UTC-3:30 (Newfoundland)
	-6: {"Brazil", "Argentina", "Chile", "Uruguay", "Suriname"}, // UTC-3
	-4: {"Brazil"},                                             // UTC-2
	-2: {"Azores"},                                             // UTC-1
	0:  {"United Kingdom", "Ireland", "Portugal", "Ghana",
		"Gambia", "Senegal", "Guinea", "Sierra Leone",
		"Liberia", "Iceland"},                                  // UTC+0
	2:  {"United Kingdom", "France", "Germany", "Netherlands",
		"Belgium", "Spain", "Italy", "Nigeria", "Tunisia",
		"Algeria", "Morocco", "Cameroon", "Niger"},             // UTC+1
	4:  {"South Africa", "Zimbabwe", "Zambia", "Kenya",
		"Tanzania", "Egypt", "Greece", "Finland",
		"Ukraine", "Romania", "Israel"},                        // UTC+2
	6:  {"Kenya", "Ethiopia", "Somalia", "Saudi Arabia",
		"Iraq", "Turkey", "Russia (Moscow)"},                   // UTC+3
	7:  {"India"},                                              // UTC+3:30
	8:  {"UAE", "Oman", "Azerbaijan"},                          // UTC+4
	9:  {"Pakistan"},                                           // UTC+4:30
	10: {"Pakistan", "Kazakhstan"},                             // UTC+5
	11: {"India", "Sri Lanka"},                                 // UTC+5:30
	12: {"Bangladesh", "Nepal"},                                // UTC+6
	14: {"Thailand", "Vietnam", "Indonesia (West)", "Cambodia"}, // UTC+7
	16: {"Singapore", "Malaysia", "Philippines", "China",
		"Hong Kong", "Taiwan", "Indonesia (Central)"},          // UTC+8
	18: {"Japan", "South Korea"},                               // UTC+9
	20: {"Australia (East)"},                                   // UTC+10
	22: {"Australia (AEDT)"},                                   // UTC+11
	24: {"New Zealand"},                                        // UTC+12
}

// activityScore returns a score for a given local hour (0-23) based on
// typical human posting behaviour. Higher = more likely to post.
func activityScore(localHour int) float64 {
	h := ((localHour % 24) + 24) % 24
	switch {
	case h >= 18 && h <= 23: // prime evening
		return 3.0
	case h >= 12 && h <= 17: // afternoon
		return 2.0
	case h >= 7 && h <= 11: // morning
		return 1.0
	case h == 0 || h == 1: // late night (still plausible)
		return 0.5
	default: // dead zone 02:00-06:00
		return -1.5
	}
}

// PostingPattern holds the results of posting-time analysis.
type PostingPattern struct {
	TotalVideos    int
	HourHistogram  [24]int   // posts per UTC hour
	BestOffsetHalf int       // winning UTC offset in half-hours (e.g. -10 = UTC-5)
	BestOffsetStr  string    // human-readable, e.g. "UTC-5"
	FitScore       float64   // higher = more confident
	CandidateTZs   []TZCandidate
	LikelyCountries []string
}

// TZCandidate is one ranked timezone.
type TZCandidate struct {
	OffsetHalf int
	OffsetStr  string
	Score      float64
	Countries  []string
}

// AnalyzePostingTimes infers timezone from video timestamps.
// Returns nil if there are fewer than 3 videos (not enough data).
func AnalyzePostingTimes(videos []VideoItem) *PostingPattern {
	var timestamps []int64
	for _, v := range videos {
		if v.CreateTime > 0 {
			timestamps = append(timestamps, v.CreateTime)
		}
	}
	if len(timestamps) < 3 {
		return nil
	}

	pp := &PostingPattern{TotalVideos: len(timestamps)}

	// Build UTC hour histogram
	for _, ts := range timestamps {
		utcHour := int((ts / 3600) % 24)
		pp.HourHistogram[utcHour]++
	}

	// Score only valid standard-timezone offsets (keys in tzCountries).
	// This avoids ties on non-standard half-hour values.
	type offsetScore struct {
		half  int
		score float64
	}
	var scores []offsetScore

	for offsetHalf := range tzCountries {
		var s float64
		for utcH := 0; utcH < 24; utcH++ {
			if pp.HourHistogram[utcH] == 0 {
				continue
			}
			localH := utcH*2 + offsetHalf // in half-hours
			localHour := ((localH/2)%24 + 24) % 24
			s += activityScore(localHour) * float64(pp.HourHistogram[utcH])
		}
		scores = append(scores, offsetScore{offsetHalf, s})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	pp.BestOffsetHalf = scores[0].half
	pp.BestOffsetStr = halfToUTCString(scores[0].half)
	pp.FitScore = scores[0].score

	// Normalise fit score to [0,1] for confidence estimation
	minS, maxS := scores[len(scores)-1].score, scores[0].score
	if maxS > minS {
		pp.FitScore = (scores[0].score - minS) / (maxS - minS)
	}

	// Collect top-5 candidates (within 25% of best score)
	threshold := scores[0].score * 0.75
	for _, os := range scores {
		if os.score < threshold || len(pp.CandidateTZs) >= 5 {
			break
		}
		countries := tzCountries[os.half]
		pp.CandidateTZs = append(pp.CandidateTZs, TZCandidate{
			OffsetHalf: os.half,
			OffsetStr:  halfToUTCString(os.half),
			Score:      os.score,
			Countries:  countries,
		})
	}

	// Flatten countries from the best candidate
	if len(pp.CandidateTZs) > 0 {
		pp.LikelyCountries = pp.CandidateTZs[0].Countries
	}
	return pp
}

// halfToUTCString converts a half-hour offset integer to "UTC±H:MM".
func halfToUTCString(half int) string {
	sign := "+"
	if half < 0 {
		sign = "-"
		half = -half
	}
	h := half / 2
	m := (half % 2) * 30
	if m == 0 {
		return fmt.Sprintf("UTC%s%d", sign, h)
	}
	return fmt.Sprintf("UTC%s%d:%02d", sign, h, m)
}

// TZWeight returns how strongly to weight the timezone signal.
// More videos and higher fit = stronger weight.
func TZWeight(pp *PostingPattern) float64 {
	// base weight increases with number of data points (diminishing returns)
	nFactor := math.Log1p(float64(pp.TotalVideos)) / math.Log1p(30)
	return 4.0 * nFactor * pp.FitScore
}

// PeakUTCHour returns the UTC hour with the most posts.
func PeakUTCHour(pp *PostingPattern) int {
	peak, pkH := 0, 0
	for h, cnt := range pp.HourHistogram {
		if cnt > peak {
			peak = cnt
			pkH = h
		}
	}
	return pkH
}

// HourHistogramStr renders a compact ASCII bar chart of posting hours.
func HourHistogramStr(pp *PostingPattern) string {
	max := 1
	for _, c := range pp.HourHistogram {
		if c > max {
			max = c
		}
	}
	var sb strings.Builder
	for h := 0; h < 24; h++ {
		bar := int(float64(pp.HourHistogram[h]) / float64(max) * 8)
		sb.WriteString(fmt.Sprintf("  %02d:00  %s (%d)\n",
			h, strings.Repeat("█", bar)+strings.Repeat("░", 8-bar), pp.HourHistogram[h]))
	}
	return sb.String()
}
