package tiktok

// AudienceCluster holds commenter-region analysis results.
type AudienceCluster struct {
	TotalCommenters int
	TopRegions      []RegionCount
	DominantISO     string  // ISO code of top region
	DominantPct     float64 // fraction [0,1] of top region
}

// RegionCount pairs an ISO country code with a commenter count.
type RegionCount struct {
	ISO     string
	Country string
	Count   int
	Pct     float64
}

// BuildAudienceCluster summarises a region→count map into an AudienceCluster.
func BuildAudienceCluster(regions map[string]int) *AudienceCluster {
	if len(regions) == 0 {
		return nil
	}
	total := 0
	for _, cnt := range regions {
		total += cnt
	}
	if total == 0 {
		return nil
	}

	ac := &AudienceCluster{TotalCommenters: total}

	for iso, cnt := range regions {
		pct := float64(cnt) / float64(total)
		ac.TopRegions = append(ac.TopRegions, RegionCount{
			ISO:     iso,
			Country: isoToCountry(iso),
			Count:   cnt,
			Pct:     pct,
		})
	}

	// Sort descending by count
	for i := 1; i < len(ac.TopRegions); i++ {
		for j := i; j > 0 && ac.TopRegions[j].Count > ac.TopRegions[j-1].Count; j-- {
			ac.TopRegions[j], ac.TopRegions[j-1] = ac.TopRegions[j-1], ac.TopRegions[j]
		}
	}

	if len(ac.TopRegions) > 0 {
		ac.DominantISO = ac.TopRegions[0].ISO
		ac.DominantPct = ac.TopRegions[0].Pct
	}
	return ac
}
