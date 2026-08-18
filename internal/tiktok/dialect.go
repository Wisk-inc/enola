package tiktok

import (
	"strings"
)

// DialectScore holds per-dialect word-match scores.
type DialectScore struct {
	Dialect string
	Score   float64
	Matches []string
}

// dialectWord is a scored vocabulary item for a dialect.
type dialectWord struct {
	word   string
	weight float64
}

// dialectProfiles lists scored vocabulary for each dialect/variety.
// Weights reflect how uniquely the word identifies the dialect.
var dialectProfiles = map[string][]dialectWord{
	// ── Jamaican Patois ───────────────────────────────────────────────────
	"Jamaican Patois": {
		// Extremely distinctive — only in Patois
		{"wagwan", 5}, {"wha gwaan", 5}, {"whagwan", 5},
		{"bredren", 4}, {"bredda", 4}, {"gyalis", 4}, {"gyallis", 4},
		{"pickney", 5}, {"jancrow", 5}, {"dutty wine", 5},
		{"jah bless", 4}, {"give thanks", 4}, {"rastafari", 4},
		{"likkle more", 5}, {"nuff respect", 4},
		{"no sah", 5}, {"lawd", 4}, {"lawdamercy", 5},
		{"sketel", 5}, {"wutless", 5},
		// Very strong
		{"mi nuh", 4}, {"mi deh", 4}, {"mi a", 4},
		{"yuh nuh", 4}, {"nuh lie", 4}, {"nah lie", 4},
		{"a suh", 4}, {"suh it go", 4},
		{"bless up", 3}, {"biggup", 3}, {"big up", 3},
		{"one love", 3}, {"irie", 4}, {"rasta", 3},
		{"dutty", 3}, {"inna", 3}, {"likkle", 4},
		{"badman", 3}, {"badgyal", 4}, {"link mi", 3},
		// Moderate — shared with other Caribbean dialects but skewed Jamaican
		{"bare gyal", 4}, {"bare man", 3},
		{"di road", 3}, {"road code", 3},
		{"jah", 3},
	},

	// ── Nigerian Pidgin English ───────────────────────────────────────────
	"Nigerian Pidgin": {
		{"abeg", 5}, {"oga", 5}, {"wahala", 5}, {"wetin", 5},
		{"dey go", 4}, {"no be", 4}, {"na so", 5}, {"na im", 4},
		{"werey", 5}, {"nna", 4}, {"nnam", 4},
		{"sharp sharp", 4}, {"make i", 4}, {"e be like", 4},
		{"enter one chance", 5}, {"maga", 4},
		{"chop", 3}, {"scatter", 3},
		{"sabi", 4}, {"kolo", 4}, {"mumu", 4},
		{"oya", 4}, {"chai", 4},
	},

	// ── Nigerian Standard English markers ────────────────────────────────
	"Nigerian English": {
		{"sha", 3}, {"joor", 4}, {"abi", 3}, {"shey", 4},
		{"how far", 3}, {"how body", 4},
		{"gist", 3}, {"hustling", 2}, {"manage", 2},
		{"naija", 5}, {"9ja", 5},
	},

	// ── UK Multicultural London English / Grime / Drill ──────────────────
	"UK English": {
		{"innit", 4}, {"bruv", 4}, {"mandem", 5}, {"roadman", 5},
		{"peng", 4}, {"wasteman", 5}, {"peak", 3},
		{"bare", 3}, {"init", 3},
		{"fam", 3}, {"blud", 4}, {"cuz", 2},
		{"mazza", 4}, {"piff", 4}, {"chirpsing", 5},
		{"yard ting", 4}, {"link up", 3},
		{"allow it", 4}, {"butters", 4}, {"buff", 3},
		{"galdem", 5}, {"mandem", 5}, {"pagans", 4},
	},

	// ── Caribbean Creole (cross-island) ──────────────────────────────────
	"Caribbean Creole": {
		{"wining", 4}, {"liming", 4}, {"tabanca", 5},
		{"allyuh", 5}, {"dotish", 5}, {"ole talk", 4},
		{"mamaguy", 5}, {"gyul", 3},
		{"soca", 4}, {"fete", 4}, {"mas", 3},
		{"bajan", 5}, {"trini", 5}, {"trinbago", 5},
		{"mudland", 5}, {"guyanese", 4},
	},

	// ── AAVE (African American Vernacular English) ────────────────────────
	"AAVE": {
		{"finna", 5}, {"bussin", 5}, {"deadass", 5},
		{"lowkey", 3}, {"no cap", 5}, {"on god", 4},
		{"slay", 3}, {"period", 3}, {"bestie", 2},
		{"it's giving", 5}, {"rent free", 4}, {"understood the assignment", 5},
		{"caught in 4k", 5}, {"main character", 4},
		{"sus", 3}, {"hit different", 4},
		{"fr fr", 4}, {"ong", 4},
	},

	// ── Ghanaian English markers ──────────────────────────────────────────
	"Ghanaian English": {
		{"chale", 5}, {"tweaa", 5}, {"ei", 3}, {"yawa", 4},
		{"wo hu", 5}, {"obroni", 5}, {"dawg", 2},
		{"akwaaba", 5}, {"ebe", 4},
	},
}

// dialectToCountry maps dialect names to the most likely country.
var dialectToCountry = map[string]string{
	"Jamaican Patois":   "Jamaica",
	"Nigerian Pidgin":   "Nigeria",
	"Nigerian English":  "Nigeria",
	"UK English":        "United Kingdom",
	"Caribbean Creole":  "", // ambiguous — handled separately
	"AAVE":              "United States",
	"Ghanaian English":  "Ghana",
}

// ScoreDialect analyses text and returns all dialect scores, sorted best-first.
func ScoreDialect(text string) []DialectScore {
	lower := strings.ToLower(text)
	var results []DialectScore

	for dialect, words := range dialectProfiles {
		var score float64
		var matches []string
		for _, dw := range words {
			if strings.Contains(lower, dw.word) {
				score += dw.weight
				matches = append(matches, dw.word)
			}
		}
		if score > 0 {
			results = append(results, DialectScore{
				Dialect: dialect,
				Score:   score,
				Matches: matches,
			})
		}
	}

	// Sort highest score first
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].Score > results[j-1].Score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}
	return results
}

// DialectCountry returns the most likely country for the best-scoring dialect,
// or "" if below the minimum threshold or ambiguous.
func DialectCountry(scores []DialectScore) (country string, evidence string, weight float64) {
	if len(scores) == 0 || scores[0].Score < 4.0 {
		return "", "", 0
	}
	best := scores[0]
	country = dialectToCountry[best.Dialect]
	if country == "" {
		return "", "", 0
	}
	evidence = best.Dialect + " markers: " + strings.Join(best.Matches, ", ")
	// Weight scales with score but is capped at 6.0
	weight = best.Score * 0.3
	if weight > 6.0 {
		weight = 6.0
	}
	return country, evidence, weight
}
