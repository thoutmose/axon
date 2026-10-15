package digest

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"strings"
)

// Renderers turn a digest into what each channel shows. base is Axon's
// address (AXON_BASE_URL) for links; without it there are none.

func link(base, path string) string {
	if base == "" || path == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + path
}

func (d Digest) tr(s string) string {
	if v, ok := words[d.Lang][s]; ok {
		return v
	}
	return s
}

func itemLine(it Item) string {
	s := it.Text
	if it.N > 0 {
		s = fmt.Sprintf("%d. %s", it.N, s)
	} else {
		s = "• " + s
	}
	if it.Detail != "" {
		s += " — " + it.Detail
	}
	return s
}

// Text is the digest as plain text: email's text part, and chat apps
// that show text only.
func Text(d Digest, base string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s — %s\n%s\n", d.Title, d.Label, d.Greeting)
	if d.Summary != "" {
		fmt.Fprintf(&b, "%s\n", d.Summary)
	}
	if len(d.Stats) > 0 {
		b.WriteString("\n")
		for _, s := range d.Stats {
			fmt.Fprintf(&b, "%s %s: %s\n", s.Emoji, s.Label, s.Value)
		}
	}
	for _, s := range d.Sections {
		fmt.Fprintf(&b, "\n%s %s\n", s.Emoji, s.Title)
		for _, it := range s.Items {
			b.WriteString(itemLine(it) + "\n")
		}
		if s.More > 0 {
			fmt.Fprintf(&b, "  "+d.tr("and %d more")+"\n", s.More)
		}
	}
	if d.Quote.Text != "" {
		fmt.Fprintf(&b, "\n“%s” — %s\n", d.Quote.Text, d.Quote.Author)
	}
	if u := link(base, d.URL); u != "" {
		fmt.Fprintf(&b, "\n%s: %s\n", d.tr("Open Axon"), u)
	}
	return b.String()
}

// Short is the digest in one line, for a notification.
func Short(d Digest) (title, body string) {
	return d.Title + " · " + d.Label, d.Summary
}

// Markdown is the digest for ntfy and other Markdown-aware channels.
func Markdown(d Digest, base string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s**\n\n", d.Greeting)
	if d.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", d.Summary)
	}
	for _, s := range d.Stats {
		fmt.Fprintf(&b, "- %s %s: **%s**\n", s.Emoji, s.Label, s.Value)
	}
	for _, s := range d.Sections {
		fmt.Fprintf(&b, "\n**%s %s**\n", s.Emoji, s.Title)
		for _, it := range s.Items {
			line := it.Text
			if u := link(base, it.URL); u != "" {
				line = "[" + it.Text + "](" + u + ")"
			}
			if it.N > 0 {
				line = fmt.Sprintf("%d. %s", it.N, line)
			} else {
				line = "- " + line
			}
			if it.Detail != "" {
				line += " — _" + it.Detail + "_"
			}
			b.WriteString(line + "\n")
		}
	}
	if d.Quote.Text != "" {
		fmt.Fprintf(&b, "\n> %s\n> — %s\n", d.Quote.Text, d.Quote.Author)
	}
	return b.String()
}

// Telegram is the digest in Telegram's HTML subset.
func Telegram(d Digest, base string) string {
	e := html.EscapeString
	var b strings.Builder
	fmt.Fprintf(&b, "<b>%s — %s</b>\n%s\n", e(d.Title), e(d.Label), e(d.Greeting))
	if d.Summary != "" {
		fmt.Fprintf(&b, "<i>%s</i>\n", e(d.Summary))
	}
	if len(d.Stats) > 0 {
		b.WriteString("\n")
		for _, s := range d.Stats {
			fmt.Fprintf(&b, "%s %s: <b>%s</b>\n", s.Emoji, e(s.Label), e(s.Value))
		}
	}
	for _, s := range d.Sections {
		fmt.Fprintf(&b, "\n%s <b>%s</b>\n", s.Emoji, e(s.Title))
		for _, it := range s.Items {
			text := e(it.Text)
			if u := link(base, it.URL); u != "" {
				text = `<a href="` + e(u) + `">` + text + `</a>`
			}
			if it.N > 0 {
				text = fmt.Sprintf("<code>%d</code> %s", it.N, text)
			} else {
				text = "• " + text
			}
			if it.Detail != "" {
				text += " — <i>" + e(it.Detail) + "</i>"
			}
			b.WriteString(text + "\n")
		}
		if s.More > 0 {
			fmt.Fprintf(&b, "<i>%s</i>\n", e(fmt.Sprintf(d.tr("and %d more"), s.More)))
		}
	}
	if d.Quote.Text != "" {
		fmt.Fprintf(&b, "\n<blockquote>%s\n— %s</blockquote>\n", e(d.Quote.Text), e(d.Quote.Author))
	}
	if len(d.Actionable()) > 0 {
		fmt.Fprintf(&b, "\n<i>%s</i>", e(d.tr("Reply to act")))
	}
	return b.String()
}

// Slack is a Block Kit message.
func Slack(d Digest, base string, replies bool) map[string]any {
	blocks := []any{
		map[string]any{"type": "header", "text": map[string]any{"type": "plain_text", "text": d.Title + " · " + d.Label, "emoji": true}},
		map[string]any{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": "*" + slackEscape(d.Greeting) + "*\n" + slackEscape(d.Summary)}},
	}
	if len(d.Stats) > 0 {
		var fields []any
		for _, s := range d.Stats {
			fields = append(fields, map[string]any{"type": "mrkdwn", "text": s.Emoji + " *" + slackEscape(s.Value) + "*  " + slackEscape(s.Label)})
		}
		blocks = append(blocks, map[string]any{"type": "section", "fields": fields[:min(len(fields), 10)]})
	}
	for _, s := range d.Sections {
		var lines []string
		for _, it := range s.Items {
			text := slackEscape(it.Text)
			if u := link(base, it.URL); u != "" {
				text = "<" + u + "|" + text + ">"
			}
			if it.N > 0 {
				text = fmt.Sprintf("`%d` %s", it.N, text)
			} else {
				text = "• " + text
			}
			if it.Detail != "" {
				text += "  _" + slackEscape(it.Detail) + "_"
			}
			lines = append(lines, text)
		}
		if s.More > 0 {
			lines = append(lines, "_"+fmt.Sprintf(d.tr("and %d more"), s.More)+"_")
		}
		blocks = append(blocks, map[string]any{"type": "divider"},
			map[string]any{"type": "section", "text": map[string]any{"type": "mrkdwn", "text": "*" + s.Emoji + " " + slackEscape(s.Title) + "*\n" + strings.Join(lines, "\n")}})
	}
	if d.Quote.Text != "" {
		blocks = append(blocks, map[string]any{"type": "divider"},
			map[string]any{"type": "context", "elements": []any{map[string]any{"type": "mrkdwn", "text": "> _" + slackEscape(d.Quote.Text) + "_\n> — " + slackEscape(d.Quote.Author)}}})
	}
	var actions []any
	if u := link(base, d.URL); u != "" {
		actions = append(actions, map[string]any{"type": "button", "text": map[string]any{"type": "plain_text", "text": d.tr("Open Axon")}, "url": u})
	}
	if len(actions) > 0 {
		blocks = append(blocks, map[string]any{"type": "actions", "elements": actions})
	}
	if replies && len(d.Actionable()) > 0 {
		blocks = append(blocks, map[string]any{"type": "context", "elements": []any{map[string]any{"type": "mrkdwn", "text": d.tr("Reply to act")}}})
	}
	return map[string]any{"text": d.Title + " · " + d.Summary, "blocks": blocks}
}

// slackEscape keeps text from being read as Slack markup or mentions.
func slackEscape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Discord is a webhook or bot message with one embed per part; the picture,
// when the person wants it, is attached as scene.png.
func Discord(d Digest, base string, replies bool) map[string]any {
	accent := Accent(d.Theme)
	colour := int(accent.R)<<16 | int(accent.G)<<8 | int(accent.B)
	head := map[string]any{
		"title":       d.Title + " · " + d.Label,
		"description": "**" + d.Greeting + "**\n" + d.Summary,
		"color":       colour,
	}
	if d.Picture {
		head["image"] = map[string]any{"url": "attachment://scene.png"}
	}
	if u := link(base, d.URL); u != "" {
		head["url"] = u
	}
	var fields []any
	for _, s := range d.Stats {
		fields = append(fields, map[string]any{"name": s.Emoji + " " + s.Label, "value": s.Value, "inline": true})
	}
	for _, s := range d.Sections {
		var lines []string
		for _, it := range s.Items {
			text := discordEscape(it.Text)
			if u := link(base, it.URL); u != "" {
				text = "[" + text + "](" + u + ")"
			}
			if it.N > 0 {
				text = fmt.Sprintf("`%d` %s", it.N, text)
			} else {
				text = "• " + text
			}
			if it.Detail != "" {
				text += " — *" + discordEscape(it.Detail) + "*"
			}
			lines = append(lines, text)
		}
		value := strings.Join(lines, "\n")
		if len(value) > 1000 { // Discord's field limit is 1024
			cut := strings.LastIndex(value[:1000], "\n")
			if cut < 0 {
				cut = strings.LastIndex(value[:1000], " ")
			}
			value = value[:max(cut, 0)] + "\n…"
		}
		fields = append(fields, map[string]any{"name": s.Emoji + " " + s.Title, "value": value})
	}
	if len(fields) > 0 {
		head["fields"] = fields[:min(len(fields), 25)]
	}
	footer := ""
	if d.Quote.Text != "" {
		footer = "“" + d.Quote.Text + "” — " + d.Quote.Author
	}
	if replies && len(d.Actionable()) > 0 {
		footer += "\n\n" + d.tr("Reply to act")
	}
	if footer = strings.TrimSpace(footer); footer != "" {
		head["footer"] = map[string]any{"text": footer}
	}
	out := map[string]any{
		"username":         "Axon",
		"embeds":           []any{head},
		"allowed_mentions": map[string]any{"parse": []any{}},
	}
	if d.Picture {
		out["attachments"] = []any{map[string]any{"id": 0, "filename": "scene.png"}}
	}
	return out
}

func discordEscape(s string) string {
	return strings.NewReplacer("*", "\\*", "_", "\\_", "`", "\\`", "~", "\\~", "|", "\\|", "[", "\\[", "]", "\\]").Replace(s)
}

// Teams is an Adaptive Card for a Teams Workflows webhook.
func Teams(d Digest, base string) map[string]any {
	body := []any{
		map[string]any{"type": "TextBlock", "text": d.Title + " · " + d.Label, "size": "Large", "weight": "Bolder", "wrap": true},
		map[string]any{"type": "TextBlock", "text": d.Greeting, "weight": "Bolder", "wrap": true, "spacing": "Small"},
	}
	if d.Summary != "" {
		body = append(body, map[string]any{"type": "TextBlock", "text": d.Summary, "isSubtle": true, "wrap": true, "spacing": "None"})
	}
	if len(d.Stats) > 0 {
		var facts []any
		for _, s := range d.Stats {
			facts = append(facts, map[string]any{"title": s.Emoji + " " + s.Label, "value": s.Value})
		}
		body = append(body, map[string]any{"type": "FactSet", "facts": facts})
	}
	for _, s := range d.Sections {
		body = append(body, map[string]any{"type": "TextBlock", "text": s.Emoji + " " + s.Title, "weight": "Bolder", "size": "Medium", "separator": true, "wrap": true})
		for _, it := range s.Items {
			text := it.Text
			if u := link(base, it.URL); u != "" {
				text = "[" + it.Text + "](" + u + ")"
			}
			if it.N > 0 {
				text = fmt.Sprintf("**%d.** %s", it.N, text)
			} else {
				text = "• " + text
			}
			if it.Detail != "" {
				text += " — _" + it.Detail + "_"
			}
			body = append(body, map[string]any{"type": "TextBlock", "text": text, "wrap": true, "spacing": "Small"})
		}
	}
	if d.Quote.Text != "" {
		body = append(body, map[string]any{"type": "TextBlock", "text": "_“" + d.Quote.Text + "”_ — " + d.Quote.Author, "wrap": true, "separator": true, "isSubtle": true})
	}
	card := map[string]any{"$schema": "http://adaptivecards.io/schemas/adaptive-card.json", "type": "AdaptiveCard", "version": "1.4", "body": body}
	if u := link(base, d.URL); u != "" {
		card["actions"] = []any{map[string]any{"type": "Action.OpenUrl", "title": d.tr("Open Axon"), "url": u}}
	}
	return map[string]any{"type": "message", "attachments": []any{map[string]any{"contentType": "application/vnd.microsoft.card.adaptive", "content": card}}}
}

// emailTmpl is a single-column email with inline styles, readable in every
// mail app, light by default and with a dark variant where supported.
var emailTmpl = template.Must(template.New("email").Funcs(template.FuncMap{
	"link": link,
}).Parse(`<!doctype html>
<html lang="{{.D.Lang}}"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width">
<meta name="color-scheme" content="light dark"><title>{{.D.Title}}</title>
<style>@media (prefers-color-scheme: dark){.card{background:#1d2230!important;color:#e8eaf0!important}.muted{color:#a9b0c2!important}.item{border-color:#2c3345!important}}</style>
</head>
<body style="margin:0;padding:0;background:#eef1f6;font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;color:#1d2433">
<table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr><td align="center" style="padding:24px 12px">
<table role="presentation" class="card" width="600" cellspacing="0" cellpadding="0" style="max-width:600px;width:100%;background:#ffffff;border-radius:16px;overflow:hidden">
{{if .CID}}<tr><td><img src="cid:{{.CID}}" width="600" alt="" style="display:block;width:100%;height:auto;border:0"></td></tr>{{end}}
<tr><td style="padding:24px 28px 8px">
<div class="muted" style="font-size:13px;letter-spacing:.06em;text-transform:uppercase;color:#6b7385">{{.D.Title}} · {{.D.Label}}</div>
<h1 style="margin:6px 0 4px;font-size:24px;line-height:1.25;color:{{.Accent}}">{{.D.Greeting}}</h1>
{{if .D.Summary}}<p class="muted" style="margin:0;font-size:15px;color:#4a5263">{{.D.Summary}}</p>{{end}}
</td></tr>
{{if .D.Stats}}<tr><td style="padding:12px 28px 4px"><table role="presentation" width="100%" cellspacing="0" cellpadding="0"><tr>
{{range .D.Stats}}<td align="center" style="padding:10px 4px;background:{{$.Soft}};border-radius:12px"><div style="font-size:20px">{{.Emoji}}</div><div style="font-size:18px;font-weight:700">{{.Value}}</div><div class="muted" style="font-size:12px;color:#6b7385">{{.Label}}</div></td><td width="6"></td>{{end}}
</tr></table></td></tr>{{end}}
{{range .D.Sections}}<tr><td style="padding:18px 28px 0">
<h2 style="margin:0 0 8px;font-size:17px">{{.Emoji}} {{.Title}}</h2>
<table role="presentation" width="100%" cellspacing="0" cellpadding="0">
{{range .Items}}<tr><td class="item" style="padding:9px 0;border-bottom:1px solid #edf0f5;font-size:15px;line-height:1.4">
{{if .N}}<span style="display:inline-block;min-width:22px;padding:1px 6px;margin-right:8px;border-radius:999px;background:{{$.Accent}};color:#fff;font-size:12px;font-weight:700;text-align:center">{{.N}}</span>{{end}}
{{with link $.Base .URL}}<a href="{{.}}" style="color:inherit;text-decoration:none">{{end}}{{.Text}}{{if link $.Base .URL}}</a>{{end}}
{{if .Detail}}<span class="muted" style="color:{{if .Late}}#c0392b{{else}}#6b7385{{end}};font-size:13px"> · {{.Detail}}</span>{{end}}
</td></tr>{{end}}
{{if .More}}<tr><td class="muted" style="padding:8px 0;font-size:13px;color:#6b7385">{{printf $.AndMore .More}}</td></tr>{{end}}
</table></td></tr>{{end}}
{{if .D.Quote.Text}}<tr><td style="padding:24px 28px">
<blockquote style="margin:0;padding:14px 18px;border-left:4px solid {{.Accent}};background:{{.Soft}};border-radius:8px;font-size:16px;line-height:1.5;font-style:italic">“{{.D.Quote.Text}}”<div class="muted" style="margin-top:6px;font-style:normal;font-size:13px;color:#6b7385">— {{.D.Quote.Author}}</div></blockquote>
</td></tr>{{end}}
{{with link .Base .D.URL}}<tr><td align="center" style="padding:0 28px 24px"><a href="{{.}}" style="display:inline-block;padding:12px 22px;border-radius:999px;background:{{$.Accent}};color:#fff;text-decoration:none;font-weight:600">{{$.Open}}</a></td></tr>{{end}}
</table>
<p class="muted" style="font-size:12px;color:#8a92a3;margin:16px 0 0">{{.Off}}</p>
</td></tr></table></body></html>`))

// HTML is the digest as an email body; cid names the inline picture, or
// is empty for none.
func HTML(d Digest, base, cid string) string {
	a := Accent(d.Theme)
	soft := mix(a, rgb(0xFFFFFF), 0.9)
	var buf bytes.Buffer
	_ = emailTmpl.Execute(&buf, map[string]any{
		"D": d, "Base": base, "CID": cid,
		"Accent":  template.CSS(fmt.Sprintf("#%02x%02x%02x", a.R, a.G, a.B)),          //nolint:gosec // a colour from the fixed palette
		"Soft":    template.CSS(fmt.Sprintf("#%02x%02x%02x", soft.R, soft.G, soft.B)), //nolint:gosec // a colour from the fixed palette
		"Open":    d.tr("Open Axon"),
		"AndMore": d.tr("and %d more"),
		"Off":     d.tr("Turn digests off"),
	})
	return buf.String()
}
