package tiktok

import (
	"fmt"
	"regexp"
	"strings"
)

// ContactInfo holds extracted contact details and their geolocation implications.
type ContactInfo struct {
	Type        string // "whatsapp", "phone", "email", "linktree"
	Raw         string // the raw string found
	PhoneNumber string // cleaned phone number if applicable
	CountryCode string // extracted country dialling code
	Country     string // mapped country (if deterministic)
	Evidence    string
	Weight      float64
}

var (
	// wa.me/+18765551234 or wa.me/18765551234
	reWaMe = regexp.MustCompile(`(?i)(?:wa\.me|whatsapp\.com/send\?phone=|api\.whatsapp\.com/send\?phone=)[/=]?\+?(\d{7,15})`)

	// WhatsApp chat invite links (no phone number, but confirms WhatsApp use)
	reWaChat = regexp.MustCompile(`(?i)chat\.whatsapp\.com/[A-Za-z0-9]+`)

	// wa.link short links (redirect to WhatsApp; number may be in URL params)
	reWaLink = regexp.MustCompile(`(?i)wa\.link/[A-Za-z0-9]+`)

	// Bare international phone numbers in text
	// Matches +1876..., +44..., +234..., etc.
	reIntlPhone = regexp.MustCompile(`(?:^|[\s(,\[])\+(\d{1,3}[\s\-.]?\d{2,5}[\s\-.]?\d{3,4}[\s\-.]?\d{3,4})`)

	// Bare local numbers with area codes in parentheses: (876) 555-1234
	reLocalPhone = regexp.MustCompile(`\((\d{3})\)\s*\d{3}[\s\-]\d{4}`)
)

// countryDialCodes maps dialling prefixes to countries. Ordered longest-first
// to avoid prefix ambiguity (e.g. +1876 before +1).
var countryDialCodes = []struct {
	prefix  string
	country string
}{
	{"1876", "Jamaica"},
	{"1868", "Trinidad and Tobago"},
	{"1246", "Barbados"},
	{"1868", "Trinidad and Tobago"},
	{"1473", "Grenada"},
	{"1784", "Saint Vincent"},
	{"1758", "Saint Lucia"},
	{"1767", "Dominica"},
	{"1869", "Saint Kitts"},
	{"1268", "Antigua"},
	{"1242", "Bahamas"},
	{"1345", "Cayman Islands"},
	{"1441", "Bermuda"},
	{"1649", "Turks and Caicos"},
	{"1284", "British Virgin Islands"},
	{"1340", "US Virgin Islands"},
	{"1787", "Puerto Rico"},
	{"1939", "Puerto Rico"},
	{"44", "United Kingdom"},
	{"234", "Nigeria"},
	{"233", "Ghana"},
	{"254", "Kenya"},
	{"27", "South Africa"},
	{"49", "Germany"},
	{"33", "France"},
	{"65", "Singapore"},
	{"60", "Malaysia"},
	{"1", "United States / Canada"},
	{"592", "Guyana"},
	{"509", "Haiti"},
	{"53", "Cuba"},
	{"1809", "Dominican Republic"},
	{"1829", "Dominican Republic"},
	{"1849", "Dominican Republic"},
}

// areaCodeCountry maps 3-digit NANP area codes that are unambiguously Caribbean.
var areaCodeCountry = map[string]string{
	"876": "Jamaica",
	"868": "Trinidad and Tobago",
	"246": "Barbados",
	"473": "Grenada",
	"784": "Saint Vincent",
	"758": "Saint Lucia",
	"767": "Dominica",
	"869": "Saint Kitts",
	"268": "Antigua",
	"242": "Bahamas",
	"345": "Cayman Islands",
}

// ExtractContacts parses all contact details from text and returns
// a slice of ContactInfo with country inferences.
func ExtractContacts(text string) []ContactInfo {
	var out []ContactInfo

	// ── wa.me links ───────────────────────────────────────────────────────
	for _, m := range reWaMe.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		number := strings.ReplaceAll(m[1], " ", "")
		number = strings.ReplaceAll(number, "-", "")
		number = strings.ReplaceAll(number, ".", "")
		country, cc := inferCountryFromNumber(number)
		ci := ContactInfo{
			Type:        "whatsapp",
			Raw:         m[0],
			PhoneNumber: "+" + number,
			CountryCode: cc,
			Country:     country,
			Weight:      6.0, // wa.me = highly reliable
		}
		if country != "" {
			ci.Evidence = fmt.Sprintf("WhatsApp number +%s → %s", number, country)
		} else {
			ci.Evidence = fmt.Sprintf("WhatsApp number +%s", number)
		}
		out = append(out, ci)
	}

	// ── WhatsApp chat links (no number, but presence confirms platform) ───
	for _, m := range reWaChat.FindAllString(text, -1) {
		out = append(out, ContactInfo{
			Type:    "whatsapp_group",
			Raw:     m,
			Evidence: "WhatsApp group invite link found",
			Weight:  1.0,
		})
	}

	// ── International phone numbers in body text ──────────────────────────
	for _, m := range reIntlPhone.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		number := cleanNumber(m[1])
		country, cc := inferCountryFromNumber(number)
		if country == "" {
			continue // skip ambiguous numbers
		}
		out = append(out, ContactInfo{
			Type:        "phone",
			Raw:         m[0],
			PhoneNumber: "+" + number,
			CountryCode: cc,
			Country:     country,
			Evidence:    fmt.Sprintf("international phone +%s → %s", number, country),
			Weight:      5.0,
		})
	}

	// ── Local area codes in parentheses: (876) 555-1234 ──────────────────
	for _, m := range reLocalPhone.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		area := m[1]
		if country, ok := areaCodeCountry[area]; ok {
			out = append(out, ContactInfo{
				Type:        "phone",
				Raw:         m[0],
				CountryCode: area,
				Country:     country,
				Evidence:    fmt.Sprintf("NANP area code (%s) → %s", area, country),
				Weight:      5.0,
			})
		}
	}

	return dedupContacts(out)
}

// inferCountryFromNumber matches a digit string (no leading +) against the
// prefix table. Returns country name and the matched prefix.
func inferCountryFromNumber(digits string) (country, prefix string) {
	for _, entry := range countryDialCodes {
		if strings.HasPrefix(digits, entry.prefix) {
			return entry.country, entry.prefix
		}
	}
	return "", ""
}

func cleanNumber(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func dedupContacts(contacts []ContactInfo) []ContactInfo {
	seen := make(map[string]struct{})
	out := contacts[:0]
	for _, c := range contacts {
		if _, ok := seen[c.Raw]; !ok {
			seen[c.Raw] = struct{}{}
			out = append(out, c)
		}
	}
	return out
}
