package watch

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

const kgArticle = `Snowflake announced it will acquire Crunchy Data for $250 million. ` +
	`The deal brings PostgreSQL to the Snowflake AI Data Cloud. Sridhar Ramaswamy, Snowflake's chief executive, said customers asked for it.`

// The model's answer keeps only what the article states: a fact quoting a
// sentence the article does not have, an unknown predicate, or a name that
// is not an entity is dropped, and an unknown type becomes a concept.
func TestParseItemKGKeepsWhatTheArticleStates(t *testing.T) {
	answer := "Here you go:\n" + `{"entities": [
		{"name": "Snowflake", "type": "organization", "class": "Data cloud company", "aliases": ["Snowflake Inc."]},
		{"name": "Crunchy Data", "type": "organization", "class": "Database company", "aliases": []},
		{"name": "PostgreSQL", "type": "database", "class": "Relational database", "aliases": ["Postgres"]},
		{"name": "Sridhar Ramaswamy", "type": "person", "class": "Chief executive", "aliases": []},
		{"name": "AI Data Cloud", "type": "spaceship", "class": "Platform", "aliases": []}],
	"facts": [
		{"subject": "Snowflake", "predicate": "acquires", "object": "Crunchy Data", "sentence": "Snowflake announced it will acquire Crunchy Data for $250 million."},
		{"subject": "Sridhar Ramaswamy", "predicate": "leads", "object": "Snowflake", "sentence": "Sridhar Ramaswamy, Snowflake’s chief executive, said customers asked for it."},
		{"subject": "Snowflake", "predicate": "acquires", "object": "PostgreSQL", "sentence": "Snowflake bought PostgreSQL outright."},
		{"subject": "Snowflake", "predicate": "loves", "object": "PostgreSQL", "sentence": "The deal brings PostgreSQL to the Snowflake AI Data Cloud."},
		{"subject": "Oracle", "predicate": "competes_with", "object": "Snowflake", "sentence": "The deal brings PostgreSQL to the Snowflake AI Data Cloud."}]}`
	kg, err := ParseItemKG(answer, kgArticle)
	if err != nil {
		t.Fatal(err)
	}
	if len(kg.Entities) != 5 || kg.Entities[4].Type != "concept" {
		t.Fatalf("entities = %+v", kg.Entities)
	}
	if len(kg.Facts) != 2 || kg.Facts[0].Predicate != "acquires" || kg.Facts[1].Subject != "Sridhar Ramaswamy" {
		t.Fatalf("facts = %+v", kg.Facts)
	}
	if _, err := ParseItemKG("I cannot help with that.", kgArticle); !errors.Is(err, ErrNotKnowledge) {
		t.Errorf("prose = %v", err)
	}
}

// Read entities join the ontology by any of their names; the others become
// entities of their own, typed and classed as read, and the facts link them.
func TestReadEntitiesJoinTheOntology(t *testing.T) {
	kg := &ItemKG{
		Entities: []KGEntity{{Name: "Iceberg", Type: "technology", Class: "Table format"}, {Name: "Crunchy Data", Type: "organization", Class: "Database company"},
			{Name: "Snowflake", Type: "organization", Class: "Data cloud company", Aliases: []string{"Snowflake Inc."}}},
		Facts: []KGFact{{Subject: "Snowflake", Predicate: "acquires", Object: "Crunchy Data", Sentence: "Snowflake will acquire Crunchy Data."},
			{Subject: "Crunchy Data", Predicate: "integrates_with", Object: "Iceberg", Sentence: "Crunchy Data reads Iceberg tables."}},
	}
	d := &gdoc{id: uuid.New(), title: "t", published: time.Now(), kg: kg}
	count, _, facts, extra := kgRead([]*gdoc{d})
	if count["apache-iceberg"] != 1 || count["snowflake"] != 1 || extra["x:crunchy data"] == nil || extra["x:crunchy data"].Class != "Database company" {
		t.Fatalf("count = %v, extra = %v", count, extra)
	}
	if facts[triple{"snowflake", "acquires", "x:crunchy data"}] == nil || facts[triple{"x:crunchy data", "integrates_with", "apache-iceberg"}] == nil {
		t.Fatalf("facts = %v", facts)
	}
}
