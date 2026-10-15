// Package audiobookshelf brings the podcasts a person downloads with their
// own Audiobookshelf server into the tech watch (EP-124): each show is a
// source, each episode an item with its show notes, and, when the operator
// runs a Whisper server, its words, read from the person's own copy.
package audiobookshelf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/thoutmose/axon/api/internal/watch"
)

// Client speaks Audiobookshelf's API with a person's API token.
type Client struct {
	Base, Token string
}

// Library is a podcast library; Show a podcast in it.
type (
	Library struct{ ID, Name string }
	Show    struct{ ID, Title string }
)

// ErrToken means Audiobookshelf refused the token.
var ErrToken = errors.New("the Audiobookshelf server refused the API token")

// ids Audiobookshelf gives items, episodes, and files.
var idRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

const (
	maxJSON  = 20 << 20
	maxAudio = 250 << 20
)

func (c *Client) base() string { return strings.TrimRight(c.Base, "/") }

func (c *Client) do(ctx context.Context, path string, limit int64) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("User-Agent", watch.UserAgent)
	res, err := watch.Client.Do(req)
	if err != nil {
		return nil, err
	}
	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		_ = res.Body.Close()
		return nil, ErrToken
	case res.StatusCode != http.StatusOK:
		_ = res.Body.Close()
		return nil, fmt.Errorf("the Audiobookshelf server answered %d", res.StatusCode)
	}
	res.Body = struct {
		io.Reader
		io.Closer
	}{io.LimitReader(res.Body, limit), res.Body}
	return res, nil
}

func (c *Client) get(ctx context.Context, path string, dst any) error {
	res, err := c.do(ctx, path, maxJSON)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	return json.NewDecoder(res.Body).Decode(dst)
}

// Libraries are the server's podcast libraries.
func (c *Client) Libraries(ctx context.Context) ([]Library, error) {
	var r struct {
		Libraries []struct {
			ID, Name, MediaType string
		} `json:"libraries"`
	}
	if err := c.get(ctx, "/api/libraries", &r); err != nil {
		return nil, err
	}
	out := []Library{}
	for _, l := range r.Libraries {
		if l.MediaType == "podcast" && idRe.MatchString(l.ID) {
			out = append(out, Library{l.ID, l.Name})
		}
	}
	return out, nil
}

// Shows are the podcasts of a library, a hundred a page.
func (c *Client) Shows(ctx context.Context, library string) ([]Show, error) {
	if !idRe.MatchString(library) {
		return nil, errors.New("not a library")
	}
	var out []Show
	for page := 0; page < 20; page++ {
		var r struct {
			Results []struct {
				ID    string `json:"id"`
				Media struct {
					Metadata struct {
						Title string `json:"title"`
					} `json:"metadata"`
				} `json:"media"`
			} `json:"results"`
			Total int `json:"total"`
		}
		if err := c.get(ctx, fmt.Sprintf("/api/libraries/%s/items?limit=100&page=%d", library, page), &r); err != nil {
			return nil, err
		}
		for _, it := range r.Results {
			if idRe.MatchString(it.ID) {
				out = append(out, Show{it.ID, it.Media.Metadata.Title})
			}
		}
		if len(r.Results) == 0 || len(out) >= r.Total {
			break
		}
	}
	return out, nil
}

type episode struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PubDate     string `json:"pubDate"`
	PublishedAt int64  `json:"publishedAt"`
	Enclosure   *struct {
		URL string `json:"url"`
	} `json:"enclosure"`
	AudioFile *struct {
		Ino      string `json:"ino"`
		Metadata struct {
			Filename string `json:"filename"`
		} `json:"metadata"`
	} `json:"audioFile"`
}

type item struct {
	Media struct {
		Metadata struct {
			Title    string `json:"title"`
			Author   string `json:"author"`
			ImageURL string `json:"imageUrl"`
		} `json:"metadata"`
		Episodes []episode `json:"episodes"`
	} `json:"media"`
}

func (c *Client) item(ctx context.Context, id string) (item, error) {
	var it item
	if !idRe.MatchString(id) {
		return it, errors.New("not an item")
	}
	return it, c.get(ctx, "/api/items/"+id+"?expanded=1", &it)
}

// Feed is a show as the tech watch reads a feed: one entry an episode,
// its link the episode's page on the server, its audio the public file
// (or that page, for an episode recorded locally).
func (c *Client) Feed(ctx context.Context, id string) (watch.Feed, error) {
	it, err := c.item(ctx, id)
	if err != nil {
		return watch.Feed{}, err
	}
	m := it.Media.Metadata
	f := watch.Feed{Title: m.Title, SiteURL: c.base() + "/item/" + id}
	image := ""
	if strings.HasPrefix(m.ImageURL, "https://") {
		image = m.ImageURL
	}
	for _, e := range it.Media.Episodes {
		if !idRe.MatchString(e.ID) {
			continue
		}
		ino := ""
		if e.AudioFile != nil && idRe.MatchString(e.AudioFile.Ino) {
			ino = e.AudioFile.Ino
		}
		page := c.base() + "/item/" + id + "?episode=" + url.QueryEscape(e.ID)
		entry := watch.Entry{GUID: "abs:" + id + ":" + e.ID + ":" + ino, URL: page, Title: e.Title, Author: m.Author,
			Summary: e.Description, Image: image, Audio: page}
		if e.Enclosure != nil && strings.HasPrefix(e.Enclosure.URL, "https://") {
			entry.Audio = e.Enclosure.URL
		}
		switch {
		case e.PublishedAt > 0:
			entry.Published = time.UnixMilli(e.PublishedAt).UTC()
		case e.PubDate != "":
			for _, layout := range []string{time.RFC1123Z, time.RFC1123} {
				if t, err := time.Parse(layout, e.PubDate); err == nil {
					entry.Published = t
					break
				}
			}
		}
		f.Items = append(f.Items, entry)
	}
	return f, nil
}

// Audio is an episode's file from the person's server, by the entry's
// guid ("abs:item:episode:ino").
func (c *Client) Audio(ctx context.Context, guid string) ([]byte, string, error) {
	parts := strings.Split(guid, ":")
	if len(parts) != 4 || parts[0] != "abs" || !idRe.MatchString(parts[1]) || !idRe.MatchString(parts[2]) || !idRe.MatchString(parts[3]) {
		return nil, "", errors.New("not an Audiobookshelf episode")
	}
	it, err := c.item(ctx, parts[1])
	if err != nil {
		return nil, "", err
	}
	name := "episode.mp3"
	for _, e := range it.Media.Episodes {
		if e.ID == parts[2] && e.AudioFile != nil && e.AudioFile.Metadata.Filename != "" {
			name = e.AudioFile.Metadata.Filename
		}
	}
	res, err := c.do(ctx, "/api/items/"+parts[1]+"/file/"+parts[3], maxAudio+1)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(res.Body)
	if len(b) > maxAudio {
		return nil, "", errors.New("the episode is larger than 250 MB")
	}
	return b, name, err
}
