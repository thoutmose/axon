package watch

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/db"
)

// The podcast of the news (EP-113): an episode script made from the
// period's best news, read aloud by the person's browser or, when the
// operator configures one, by a speech server. Nothing is generated: the
// words are the sources' own summaries, between short spoken links.

// PodcastOptions shape an episode.
type PodcastOptions struct {
	Format  string // briefing (a sentence each), digest (a few), deep (by topic, more)
	Lang    string // en, fr: the spoken links; the news stay in their language
	Minutes int    // target length, 3 to 30
	Days    int    // the period: 1 or 7
	TopicID string // optional
}

// PodcastFormats are the formats, in the order they are offered.
var PodcastFormats = []string{"briefing", "digest", "deep"}

// Chapter is one story, or a topic's heading in a deep dive.
type Chapter struct {
	Title  string     `json:"title"`
	Text   string     `json:"text"` // what is read aloud
	ItemID *uuid.UUID `json:"item_id"`
	URL    string     `json:"url"`
	Source string     `json:"source"`
	Topic  string     `json:"topic"`
	// Material is what the story's item says, for the hosts' conversation.
	Material string `json:"-"`
}

// Turn is one host's line of the conversation; Chapter is the story it is
// about, -1 for the opening and the close.
type Turn struct {
	Host    int    `json:"host"`
	Chapter int    `json:"chapter"`
	Text    string `json:"text"`
}

// Episode is a podcast episode's script.
type Episode struct {
	Title    string    `json:"title"`
	Lang     string    `json:"lang"`
	Format   string    `json:"format"`
	Intro    string    `json:"intro"`
	Chapters []Chapter `json:"chapters"`
	Outro    string    `json:"outro"`
	Words    int       `json:"words"`
	Minutes  float64   `json:"minutes"` // at 150 words a minute
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	// The conversation the person's model wrote, when it did: the hosts'
	// names and their lines, read in place of the parts above.
	Hosts []string `json:"hosts"`
	Turns []Turn   `json:"turns"`
}

// WordsPerMinute is a calm reading pace.
const WordsPerMinute = 150

type podcastWords struct {
	title, intro, introNone, topic, from, also, outro string
	months                                            [12]string
}

var podcastLang = map[string]podcastWords{
	"en": {
		title: "Tech watch, %s", intro: "Hello. Here is your tech watch for %s: %d stories, about %d minutes.",
		introNone: "Hello. There is no news in your tech watch for %s.",
		topic:     "Now, %s.", from: "From %s: %s", also: "Also reported by %s.",
		outro:  "That was your tech watch. It was put together by Axon Project from the sources you follow. Until next time.",
		months: [12]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"},
	},
	"fr": {
		title: "Veille technologique, %s", intro: "Bonjour. Voici votre veille technologique du %s : %d sujets, environ %d minutes.",
		introNone: "Bonjour. Votre veille technologique du %s ne contient aucune actualité.",
		topic:     "Passons à %s.", from: "Selon %s : %s", also: "Également repris par %s.",
		outro:  "C’était votre veille technologique, préparée par Axon Project à partir des sources que vous suivez. À bientôt.",
		months: [12]string{"janvier", "février", "mars", "avril", "mai", "juin", "juillet", "août", "septembre", "octobre", "novembre", "décembre"},
	},
}

// Check validates the options and fills the defaults.
func (o *PodcastOptions) Check() error {
	if o.Format == "" {
		o.Format = "digest"
	}
	if o.Lang == "" {
		o.Lang = "en"
	}
	if o.Minutes == 0 {
		o.Minutes = 10
	}
	if o.Days == 0 {
		o.Days = 1
	}
	switch {
	case !slices.Contains(PodcastFormats, o.Format):
		return ValidationError{"format is briefing, digest, or deep."}
	case podcastLang[o.Lang].intro == "":
		return ValidationError{"lang is en or fr."}
	case o.Minutes < 3 || o.Minutes > 30:
		return ValidationError{"minutes is between 3 and 30."}
	case o.Days != 1 && o.Days != 7:
		return ValidationError{"days is 1 or 7."}
	}
	return nil
}

var (
	speakLinks     = regexp.MustCompile(`https?://\S+|www\.\S+`)
	speakSentences = regexp.MustCompile(`[^.!?]+(?:[.!?]+|$)`)
	speakMarks     = regexp.MustCompile("[*`]+")
	speakNoise     = regexp.MustCompile(`[_#>|\[\]{}]+`)
	speakSpace     = regexp.MustCompile(` ([?!.,])`)
)

// forVoice removes what a voice should not read: links, Markdown marks,
// line breaks.
func forVoice(s string) string {
	// A sentence that points to a link ("More at https://…") says nothing
	// once the link is gone. Links are marked first: their dots are not
	// the ends of sentences.
	s = speakLinks.ReplaceAllString(s, "\x00")
	var kept strings.Builder
	for _, sen := range speakSentences.FindAllString(s, -1) {
		if !strings.Contains(sen, "\x00") {
			kept.WriteString(sen)
		}
	}
	s = speakMarks.ReplaceAllString(kept.String(), "")
	s = speakNoise.ReplaceAllString(s, " ")
	s = strings.Join(strings.Fields(s), " ")
	s = speakSpace.ReplaceAllString(s, "$1")
	return strings.TrimRight(s, " .:;,")
}

// speakable is text for a voice, ending as a sentence.
func speakable(s string) string {
	s = forVoice(s)
	if s == "" || strings.HasSuffix(s, "!") || strings.HasSuffix(s, "?") {
		return s
	}
	return s + "."
}

func words(s string) int { return len(strings.Fields(s)) }

type podcastItem struct {
	Item
	text   string // the longer text, for deep dives
	topics []string
}

// BuildEpisode writes the episode of the period ending at now.
func BuildEpisode(ctx context.Context, q db.Querier, userID uuid.UUID, o PodcastOptions, now time.Time, loc *time.Location) (Episode, error) {
	if err := o.Check(); err != nil {
		return Episode{}, err
	}
	w := podcastLang[o.Lang]
	ep := Episode{Lang: o.Lang, Format: o.Format, Chapters: []Chapter{}, To: now, From: now.AddDate(0, 0, -o.Days), Hosts: []string{}, Turns: []Turn{}}
	local := now.In(loc)
	date := fmt.Sprintf("%d %s %d", local.Day(), w.months[local.Month()-1], local.Year())
	if o.Lang == "en" {
		date = fmt.Sprintf("%s %d, %d", w.months[local.Month()-1], local.Day(), local.Year())
	}
	ep.Title = fmt.Sprintf(w.title, date)

	f := Filter{From: &ep.From, To: &ep.To, TopicID: o.TopicID, Limit: 200}
	where, args := f.where(userID)
	rows, err := q.Query(ctx, `SELECT `+itemColumns+`, left(i.content, 6000) FROM watch_items i WHERE `+where+
		` ORDER BY i.score DESC, cardinality(i.also_in) DESC, i.published_at DESC LIMIT 200`, args...)
	if err != nil {
		return ep, err
	}
	items, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (podcastItem, error) {
		var content string
		it, err := scanItem(r, &content)
		return podcastItem{Item: it, text: mdText(content)}, err
	})
	if err != nil {
		return ep, err
	}
	topics := map[string]string{}
	if ts, err := ListTopics(ctx, q, userID); err == nil {
		for _, t := range ts {
			topics[t.ID] = t.Name
		}
	}
	for i := range items {
		for _, id := range items[i].TopicIDs {
			if n, ok := topics[id]; ok {
				items[i].topics = append(items[i].topics, n)
			}
		}
	}

	budget := o.Minutes*WordsPerMinute - 60 // the intro and the outro
	chars := map[string]int{"briefing": 160, "digest": 420, "deep": 900}[o.Format]
	story := func(it podcastItem) Chapter {
		text := it.Summary
		if o.Format == "deep" && len(it.text) > len(text) {
			text = it.text
		}
		text = speakable(firstSentences(speakable(text), chars))
		spoken := fmt.Sprintf(w.from, forVoice(it.SourceTitle), speakable(it.Title))
		if text != "" && forVoice(text) != forVoice(it.Title) {
			spoken += " " + text
		}
		if len(it.AlsoIn) > 0 && o.Format != "briefing" {
			spoken += " " + fmt.Sprintf(w.also, strings.Join(it.AlsoIn[:min(2, len(it.AlsoIn))], ", "))
		}
		id := it.ID
		topic := ""
		if len(it.topics) > 0 {
			topic = it.topics[0]
		}
		material := it.Summary
		if len(it.text) > len(material) {
			material = it.text
		}
		return Chapter{Title: it.Title, Text: spoken, ItemID: &id, URL: it.URL, Source: it.SourceTitle, Topic: topic,
			Material: firstSentences(forVoice(material), 2500)}
	}
	var chosen []Chapter
	used := 0
	for _, it := range items {
		c := story(it)
		if used+words(c.Text) > budget && len(chosen) > 0 {
			continue // a shorter one may still fit
		}
		chosen = append(chosen, c)
		used += words(c.Text)
	}
	if o.Format == "deep" {
		// By topic, the topics with most stories first, each announced.
		count := map[string]int{}
		for _, c := range chosen {
			count[c.Topic]++
		}
		slices.SortStableFunc(chosen, func(a, b Chapter) int {
			if a.Topic == "" || b.Topic == "" {
				return cmp.Compare(boolInt(a.Topic == ""), boolInt(b.Topic == ""))
			}
			return cmp.Or(cmp.Compare(count[b.Topic], count[a.Topic]), strings.Compare(a.Topic, b.Topic))
		})
		for i := range chosen {
			if chosen[i].Topic != "" && (i == 0 || chosen[i-1].Topic != chosen[i].Topic) {
				chosen[i].Text = fmt.Sprintf(w.topic, chosen[i].Topic) + " " + chosen[i].Text
			}
		}
	}
	ep.Chapters = append(ep.Chapters, chosen...)
	ep.Words = used
	minutes := int(math.Round(float64(used+60) / WordsPerMinute))
	if len(chosen) == 0 {
		ep.Intro = fmt.Sprintf(w.introNone, date)
	} else {
		ep.Intro = fmt.Sprintf(w.intro, date, len(chosen), max(1, minutes))
	}
	ep.Outro = w.outro
	ep.Words += words(ep.Intro) + words(ep.Outro)
	ep.Minutes = math.Round(float64(ep.Words)/WordsPerMinute*10) / 10
	return ep, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Script is the whole episode as one text, for a speech server.
func (e Episode) Script() string {
	parts := []string{e.Intro}
	for _, c := range e.Chapters {
		parts = append(parts, c.Text)
	}
	return strings.Join(append(parts, e.Outro), "\n\n")
}
