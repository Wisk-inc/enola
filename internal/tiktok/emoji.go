package tiktok

import "strings"

// Regional Indicator Symbol Letter A starts at U+1F1E6.
// A two-character sequence of Regional Indicators encodes a flag emoji.
// e.g. 🇯🇲 = U+1F1EF (J) + U+1F1F2 (M)
const regionalIndicatorBase rune = 0x1F1E6

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

// flagToISO converts a Regional Indicator pair to an ISO 3166-1 alpha-2 code.
func flagToISO(r1, r2 rune) string {
	c1 := byte(r1-regionalIndicatorBase) + 'A'
	c2 := byte(r2-regionalIndicatorBase) + 'A'
	return string([]byte{c1, c2})
}

// ExtractFlagEmojis returns all ISO country codes found as flag emojis in text.
// e.g. "wagwan 🇯🇲🔥" → ["JM"]
func ExtractFlagEmojis(text string) []string {
	runes := []rune(text)
	seen := make(map[string]struct{})
	var out []string
	for i := 0; i < len(runes)-1; i++ {
		if isRegionalIndicator(runes[i]) && isRegionalIndicator(runes[i+1]) {
			iso := flagToISO(runes[i], runes[i+1])
			if _, ok := seen[iso]; !ok {
				seen[iso] = struct{}{}
				out = append(out, iso)
			}
			i++ // skip second half of the pair
		}
	}
	return out
}

// FlagWeight returns how strongly to weight a flag emoji signal.
// A flag in the bio is deliberate self-identification — one of the highest
// confidence signals available.
func FlagWeight(source string) float64 {
	switch {
	case strings.Contains(source, "bio"):
		return 8.0 // bio flag is near-certain
	case strings.Contains(source, "nickname"):
		return 6.0
	case strings.Contains(source, "comment"):
		return 3.0
	default:
		return 4.0
	}
}
