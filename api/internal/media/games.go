package media

import (
	"context"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Video games (EP-149) are looked up in English Wikipedia: its search says
// what each page is ("2017 video game") and has its cover art, and the page
// summary has the text and the year. The developer is read from the
// summary's first sentence when it says "developed by".

var (
	gameYear      = regexp.MustCompile(`\b(19[5-9]\d|20\d\d)\b`)
	gameDeveloper = regexp.MustCompile(`(?i)\bdeveloped (?:and published )?by ([A-Z0-9][^,.;()]{1,80}?)(?: and published by| and released| for |,|\.|;| \()`)
	gameGenre     = regexp.MustCompile(`(?i)^(?:\d{4}\s+)?(.*?)\s*(?:video\s+)?game\b`)
	// "Hades (video game)", "Doom (2016 video game)": the page's disambiguation, not the game's name.
	gameSuffix = regexp.MustCompile(`(?i)\s*\([^)]*\bgame\)$`)
)

// isGame tells a video game's page from the others by its short
// description: "2020 video game", "Factory simulation game", but not
// "Video game subgenre", "Video game series", or "Tabletop game".
func isGame(description string) bool {
	d := strings.ToLower(strings.TrimSpace(description))
	for _, no := range []string{"genre", "series", "franchise", "company", "studio", "developer", "publisher", "character", "console",
		"engine", "award", "convention", "magazine", "film", "tournament", "board", "card", "tabletop", "dice", "sport"} {
		if strings.Contains(d, no) {
			return false
		}
	}
	return strings.HasSuffix(d, " game") || d == "video game"
}

// coverKey is a picture's address on Wikipedia's own image servers, without
// its query, or "" for any other: covers come only from there.
func (c *Catalogs) coverKey(src string) string {
	if strings.HasPrefix(src, "//") {
		src = "https:" + src
	}
	src, _, _ = strings.Cut(src, "?")
	if len(src) > 200 || strings.Contains(src, "..") {
		return ""
	}
	for _, host := range c.WikimediaImages {
		if strings.HasPrefix(src, host) && len(src) > len(host) {
			return src
		}
	}
	return ""
}

// validPage is a Wikipedia page key a lookup can name.
func validPage(key string) bool {
	return key != "" && len(key) <= 100 && !strings.ContainsAny(key, "?#\\") && !strings.Contains(key, "..")
}

func gameFrom(key, title, description, extract, cover string) Candidate {
	cand := Candidate{Source: "wikipedia", SourceID: key, Kind: "game", Title: truncate(gameSuffix.ReplaceAllString(title, ""), 500), Genres: []string{},
		Overview: truncate(extract, 4000)}
	if y := gameYear.FindString(description); y != "" {
		n, _ := strconv.Atoi(y)
		cand.Year = &n
	}
	if m := gameGenre.FindStringSubmatch(description); m != nil && strings.TrimSpace(m[1]) != "" {
		cand.Genres = cleanGenres([]string{strings.TrimSpace(m[1])})
	}
	if m := gameDeveloper.FindStringSubmatch(extract); m != nil {
		cand.Creators = truncate(strings.TrimSpace(m[1]), 500)
	}
	if cover != "" {
		cand.CoverKey, cand.HasCover = cover, true
	}
	return cand
}

// searchWikipedia asks Wikipedia's search for pages with their short
// description and their page image (cover art is not free, so any licence).
// wikiPage is one search result with its short description and picture.
type wikiPage struct {
	Index       int    `json:"index"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Thumbnail   *struct {
		Source string `json:"source"`
	} `json:"thumbnail"`
}

func (c *Catalogs) searchWikipedia(ctx context.Context, query string) ([]Candidate, error) {
	var res struct {
		Query struct {
			Pages []wikiPage `json:"pages"`
		} `json:"query"`
	}
	// Cover art is not free, so page images of any licence.
	q := url.Values{"action": {"query"}, "format": {"json"}, "formatversion": {"2"}, "generator": {"search"}, "gsrsearch": {query},
		"gsrlimit": {"20"}, "prop": {"pageimages|description"}, "piprop": {"thumbnail"}, "pithumbsize": {"400"}, "pilicense": {"any"}}
	if err := c.get(ctx, "wikipedia", c.Wikipedia+"/w/api.php?"+q.Encode(), &res); err != nil {
		return nil, err
	}
	pages := res.Query.Pages
	slices.SortFunc(pages, func(a, b wikiPage) int { return a.Index - b.Index })
	out := []Candidate{}
	for _, p := range pages {
		key := strings.ReplaceAll(p.Title, " ", "_")
		if !isGame(p.Description) || !validPage(key) || len(out) == 10 {
			continue
		}
		img := ""
		if p.Thumbnail != nil {
			img = c.coverKey(p.Thumbnail.Source)
		}
		out = append(out, gameFrom(key, p.Title, p.Description, "", img))
	}
	return out, nil
}

func (c *Catalogs) wikipediaGame(ctx context.Context, key string) (Candidate, error) {
	var res struct {
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Extract     string `json:"extract"`
		Original    *struct {
			Source string `json:"source"`
		} `json:"originalimage"`
		Thumbnail *struct {
			Source string `json:"source"`
		} `json:"thumbnail"`
	}
	if err := c.get(ctx, "wikipedia", c.Wikipedia+"/api/rest_v1/page/summary/"+url.PathEscape(key), &res); err != nil {
		return Candidate{}, err
	}
	if res.Title == "" || res.Type == "disambiguation" || !isGame(res.Description) {
		return Candidate{}, ValidationError{"That catalogue entry does not exist."}
	}
	img := ""
	if res.Thumbnail != nil {
		img = res.Thumbnail.Source
	} else if res.Original != nil {
		img = res.Original.Source
	}
	return gameFrom(key, res.Title, res.Description, res.Extract, c.coverKey(img)), nil
}
