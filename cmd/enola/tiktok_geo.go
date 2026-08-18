package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/theyahya/enola/internal/tiktok"
	"github.com/spf13/cobra"
)

var tiktokGeoCmd = &cobra.Command{
	Use:   "tiktok-geo {username|url}",
	Short: "Geolocate a TikTok account by username or URL",
	Long: `Searches the TikTok profile, linked pages, Gmail addresses, and the web
to make an intelligent guess about which country an account is based in.
Tuned for Jamaica and the wider Caribbean region.`,
	Args: cobra.ExactArgs(1),
	Run:  runTikTokGeo,
}

func init() {
	rootCmd.AddCommand(tiktokGeoCmd)
}

func runTikTokGeo(_ *cobra.Command, args []string) {
	input := args[0]

	fmt.Printf("\n[*] Starting TikTok geolocation analysis for: %s\n\n", input)

	locator := tiktok.NewGeoLocator()
	result, err := locator.Analyze(context.Background(), input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	printGeoResult(result)
}

func printGeoResult(r *tiktok.GeoResult) {
	sep := strings.Repeat("─", 60)

	fmt.Printf("%s\n", sep)
	fmt.Printf("  TikTok Geolocation Report\n")
	fmt.Printf("%s\n\n", sep)

	fmt.Printf("  Username  : @%s\n", r.Username)
	fmt.Printf("  URL       : %s\n", r.TikTokURL)

	if r.Profile != nil {
		if r.Profile.Nickname != "" {
			fmt.Printf("  Nickname  : %s\n", r.Profile.Nickname)
		}
		if r.Profile.Bio != "" {
			fmt.Printf("  Bio       : %s\n", r.Profile.Bio)
		}
		if r.Profile.Region != "" {
			fmt.Printf("  Region    : %s\n", r.Profile.Region)
		}
	}

	fmt.Println()

	// ── Country Scores ────────────────────────────────────────────────────
	if len(r.CountryScores) > 0 {
		fmt.Printf("%s\n", sep)
		fmt.Println("  Country Probability Scores")
		fmt.Printf("%s\n", sep)

		type cs struct {
			country string
			score   float64
		}
		var ranked []cs
		var total float64
		for c, s := range r.CountryScores {
			ranked = append(ranked, cs{c, s})
			total += s
		}
		sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })

		for i, item := range ranked {
			pct := item.score / total * 100
			bar := strings.Repeat("█", int(pct/5))
			marker := "  "
			if i == 0 {
				marker = "→ "
			}
			fmt.Printf("  %s%-20s %5.1f%%  %s\n", marker, item.country, pct, bar)
		}
		fmt.Println()
	}

	// ── Best Guess ───────────────────────────────────────────────────────
	fmt.Printf("%s\n", sep)
	fmt.Printf("  Best Guess : %s\n", r.BestGuess)
	fmt.Printf("  Confidence : %s\n", r.Confidence)
	fmt.Printf("%s\n\n", sep)

	// ── Evidence Signals ─────────────────────────────────────────────────
	if len(r.Signals) > 0 {
		fmt.Println("  Evidence Signals")
		fmt.Printf("%s\n", sep)
		// Group by country
		byCountry := make(map[string][]tiktok.Signal)
		for _, s := range r.Signals {
			byCountry[s.Country] = append(byCountry[s.Country], s)
		}
		// Print best country first
		countries := make([]string, 0, len(byCountry))
		for c := range byCountry {
			countries = append(countries, c)
		}
		sort.Slice(countries, func(i, j int) bool {
			return r.CountryScores[countries[i]] > r.CountryScores[countries[j]]
		})
		for _, c := range countries {
			fmt.Printf("\n  [%s]\n", c)
			seen := make(map[string]bool)
			for _, s := range byCountry[c] {
				key := s.Evidence + "|" + s.Source
				if seen[key] {
					continue
				}
				seen[key] = true
				fmt.Printf("    • %s  (source: %s, weight: %.1f)\n", s.Evidence, s.Source, s.Weight)
			}
		}
		fmt.Println()
	}

	// ── Linked URLs ───────────────────────────────────────────────────────
	if len(r.LinkedURLs) > 0 {
		fmt.Printf("%s\n", sep)
		fmt.Println("  Linked / Discovered URLs")
		fmt.Printf("%s\n", sep)
		for _, u := range r.LinkedURLs {
			fmt.Printf("    • %s\n", u)
		}
		fmt.Println()
	}

	// ── Email Addresses ───────────────────────────────────────────────────
	if len(r.Emails) > 0 {
		fmt.Printf("%s\n", sep)
		fmt.Println("  Discovered Email Addresses")
		fmt.Printf("%s\n", sep)
		seen := make(map[string]bool)
		for _, e := range r.Emails {
			if seen[e.Email] {
				continue
			}
			seen[e.Email] = true
			fmt.Printf("\n  Gmail: %s\n", e.Email)
			if len(e.FoundAt) > 0 {
				fmt.Printf("    Found at  : %s\n", strings.Join(e.FoundAt, ", "))
			}
			if len(e.LinkedAccounts) > 0 {
				fmt.Println("    Linked accounts:")
				for _, la := range e.LinkedAccounts {
					fmt.Printf("      • %s\n", la)
				}
			}
			if len(e.Clues) > 0 {
				fmt.Println("    Location clues:")
				for i, clue := range e.Clues {
					if i >= 3 {
						break
					}
					clue = strings.ReplaceAll(clue, "\n", " ")
					if len(clue) > 120 {
						clue = clue[:117] + "..."
					}
					fmt.Printf("      » %s\n", clue)
				}
			}
		}
		fmt.Println()
	}

	if len(r.CountryScores) == 0 {
		fmt.Println("  [!] No geolocation signals found. Try a different username or add more linked accounts.")
	}
}
