package vault

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/graph"
	"github.com/thoutmose/axon/api/internal/notes"
)

// Folders is the structure of a new vault: an inbox, PARA, the
// Zettelkasten, daily notes, templates, and attachments. They are made on
// the server, empty; Axon keeps no folders, only paths.
var Folders = []string{"0-Inbox", "1-Projects", "2-Areas", "3-Resources", "4-Archive", "Zettel/literature", "Zettel/permanent", "Daily", "Templates", "Attachments"}

// templates.json is the web app's built-in templates as vault files
// (builtInFiles in web/src/lib/noteTemplates.ts); a web test keeps the two equal.
//
//go:embed templates.json
var templatesJSON []byte

type file struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// home is the note at the top of a new vault: what each folder is for.
const home = `---
type: resource
tags: [axon]
---
# Home

This vault is kept in step with Axon: edit a note here or there and the other follows.

- **0-Inbox** — fleeting notes, to process later.
- **1-Projects** — outcomes with a deadline, one folder each.
- **2-Areas** — responsibilities you keep up, with no end date.
- **3-Resources** — topics and references you are interested in.
- **4-Archive** — what is finished or paused.
- **Zettel** — literature notes (what a source says) and permanent notes (one idea each).
- **Daily** — one note a day.
- **Templates** — the templates Axon offers when you create a note; edit them, add your own.
- **Axon** — your tasks, goals, habits, projects, and media as notes, kept in step with Axon.
`

// Scaffold adds the home note and the templates to Axon where their paths
// are free, and returns the paths it added. The next sync writes them.
func Scaffold(ctx context.Context, tx pgx.Tx, userID uuid.UUID) ([]string, error) {
	var files []file
	if err := json.Unmarshal(templatesJSON, &files); err != nil {
		return nil, err
	}
	files = append([]file{{Path: "Home.md", Content: home}}, files...)
	added := []string{}
	for _, f := range files {
		// Look before creating: a failed insert would end the transaction.
		_, err := notes.GetByPath(ctx, tx, userID, f.Path)
		if err == nil {
			continue
		}
		if !errors.Is(err, graph.ErrNotFound) {
			return nil, err
		}
		if _, err := notes.Create(ctx, tx, userID, graph.UserActor, f.Path, f.Content); err != nil {
			return nil, err
		}
		added = append(added, f.Path)
	}
	return added, nil
}
