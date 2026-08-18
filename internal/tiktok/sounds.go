package tiktok

import (
	"encoding/json"
	"strings"
)

// ArtistOrigin maps a country to a list of well-known artists from there.
// Used to score TikTok video sounds (music tracks).
var artistsByCountry = map[string][]string{
	"Jamaica": {
		// Dancehall modern era
		"vybz kartel", "kartel", "addi world", "worl boss", "teacha",
		"popcaan", "unruly boss",
		"skillibeng", "skilli",
		"shenseea",
		"spice",
		"alkaline", "neutral zone",
		"mavado", "gully god",
		"masicka",
		"valiant",
		"daddy1",
		"shane o",
		"teejay",
		"aidonia",
		"busy signal",
		"tommy lee sparta", "tommy lee",
		"jada kingdom",
		"demarco",
		"gage",
		"rytikal",
		"intence",
		"chronic law",
		"jahvillani",
		"govana",
		"d'angel",
		"mr vegas",
		"i-octane",
		// Reggae / Roots
		"chronixx",
		"protoje",
		"koffee",
		"jah9",
		"mortimer",
		"lila ike",
		"sevana",
		"romain virgo",
		"christopher martin",
		"richie spice",
		"luciano",
		"capleton",
		"sizzla",
		"buju banton",
		"beres hammond",
		"sanchez",
		"freddie mcgregor",
		"beenie man",
		"sean paul",
		"shabba ranks",
		"bounty killer",
		"junior gong", "damian marley",
		"stephen marley",
		"ziggy marley",
		"bob marley",
		"toots and the maytals", "toots hibbert",
		"burning spear",
		"dennis brown",
		"gregory isaacs",
		"barrington levy",
		"yellowman",
		"super cat",
		"ninja man",
		"admiral tibet",
		"sugar minott",
	},
	"Trinidad and Tobago": {
		"machel montano", "machel",
		"bunji garlin",
		"destra garcia", "destra",
		"patrice roberts",
		"super blue",
		"nicki minaj",  // Trinidadian-American
		"kes the band", "kes",
		"fay ann lyons",
		"david rudder",
		"lord kitchener",
		"mighty sparrow",
		"calypso rose",
		"5ive",
		"voice",
		"kernal roberts",
	},
	"Nigeria": {
		"burna boy",
		"wizkid", "starboy",
		"davido",
		"tiwa savage",
		"rema",
		"tems",
		"fireboy dml", "fireboy",
		"ckay",
		"oxlade",
		"ayra starr",
		"omah lay",
		"joeboy",
		"bad boy timz",
		"naira marley", "marley",
		"zlatan ibile", "zlatan",
		"olamide",
		"dbanj", "d'banj",
		"2baba", "2face",
		"p-square",
		"flavour",
		"fela kuti",
		"asa",
		"yemi alade",
		"mr eazi",
		"runtown",
		"simi",
		"falz",
		"phyno",
	},
	"Ghana": {
		"shatta wale",
		"stonebwoy",
		"sarkodie",
		"r2bees",
		"efya",
		"king promise",
		"kidi",
		"kuami eugene",
		"medikal",
		"fameye",
		"kofi kinaata",
		"gyakie",
		"wendy shay",
		"ebony reigns",
		"tinny",
	},
	"United Kingdom": {
		"stormzy",
		"dave",
		"headie one",
		"digga d",
		"skepta", "jme", "grime",
		"giggs",
		"aitch",
		"central cee",
		"little simz",
		"j hus",
		"kano",
		"wretch 32",
		"wiley",
		"dizzee rascal",
		"lethal bizzle",
		"chip",
		"hardy caprio",
		"nines",
		"Not3s",
		"mostack",
	},
	"Barbados": {
		"rihanna",
		"cover drive",
	},
}

// SoundAPIResponse is the TikTok API response for a video's music/sound info.
type SoundAPIResponse struct {
	Music struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		AuthorName string `json:"authorName"`
		Album      string `json:"album"`
	} `json:"music"`
}

// ScoreSoundArtist checks an artist name or sound title against the artist
// database and returns the country and evidence string.
// Both the authorName and the sound title are checked.
func ScoreSoundArtist(authorName, title string) (country, evidence string, weight float64) {
	candidates := []string{
		strings.ToLower(authorName),
		strings.ToLower(title),
	}

	for c, artists := range artistsByCountry {
		for _, artist := range artists {
			for _, candidate := range candidates {
				if strings.Contains(candidate, artist) {
					field := "sound author"
					if !strings.Contains(strings.ToLower(authorName), artist) {
						field = "sound title"
					}
					return c,
						"known " + c + " artist in " + field + ": " + artist,
						3.5
				}
			}
		}
	}
	return "", "", 0
}

// extractSoundFromVideoJSON parses TikTok's embedded video JSON for sound info.
func extractSoundFromVideoJSON(rawJSON string) (authorName, title string) {
	// TikTok embeds music info as {"music":{"authorName":"...","title":"..."}}
	// Try a quick JSON parse of fragments
	var resp SoundAPIResponse
	if err := json.Unmarshal([]byte(rawJSON), &resp); err == nil {
		return resp.Music.AuthorName, resp.Music.Title
	}
	// Regex fallback
	authorName = extractJSONString(rawJSON, "authorName")
	if authorName == "" {
		authorName = extractJSONString(rawJSON, "musicAuthor")
	}
	title = extractJSONString(rawJSON, "musicName")
	if title == "" {
		title = extractJSONString(rawJSON, "title")
	}
	return
}

// extractJSONString extracts the first value of a JSON string key using a regex
// (avoids a full JSON parse on large HTML blobs).
func extractJSONString(body, key string) string {
	needle := `"` + key + `":"` // "key":"
	idx := strings.Index(body, needle)
	if idx == -1 {
		return ""
	}
	start := idx + len(needle)
	end := strings.Index(body[start:], `"`)
	if end == -1 {
		return ""
	}
	return unescapeJSON(body[start : start+end])
}
