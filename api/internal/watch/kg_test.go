package watch

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/thoutmose/axon/api/internal/knowledge"
)

// Names resolve to one entity, whatever alias the news uses, and names
// that are common words only count with their case.
func TestTheOntologyNamesEntities(t *testing.T) {
	for _, c := range []struct {
		sentence string
		want     []string
	}{
		{"Postgres 18 and K8s operators", []string{"postgresql", "kubernetes"}},
		{"Kafka/Flink pipelines on AWS", []string{"apache-kafka", "apache-flink", "amazon-web-services"}},
		{"Teams go back to the office; Go 1.26 ships", []string{"go"}},
		{"Patch CVE-2026-12345 in OpenSSH now", []string{"cve:CVE-2026-12345"}},
		{"Node.js and C++ and C# walk into a bar", []string{"node-js", "cplusplus", "csharp"}},
	} {
		var got []string
		for _, m := range onto.mentions(c.sentence) {
			got = append(got, m.id)
		}
		if len(got) != len(c.want) {
			t.Errorf("%q: %v, want %v", c.sentence, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%q: %v, want %v", c.sentence, got, c.want)
				break
			}
		}
	}
}

// Facts are read between two entities next to each other, the right way
// round, and every fact keeps the news it was read in.
func TestFactsAreReadFromTheNews(t *testing.T) {
	d := &gdoc{id: uuid.New(), title: "Data news", published: time.Now(), body: "IBM acquires Confluent for $11 billion. " +
		"Snowflake was acquired by Salesforce in a rumour nobody believed. " +
		"OpenTofu adds support for Apache Iceberg. Uber migrated from MySQL to PostgreSQL last year. " +
		"A flaw in Kubernetes is tracked as CVE-2026-0001. Teams compare Dagster vs Apache Airflow."}
	_, _, facts, _ := kgRead([]*gdoc{d})
	for _, want := range []triple{
		{"ibm", "acquires", "confluent"},
		{"salesforce", "acquires", "snowflake"},
		{"opentofu", "integrates_with", "apache-iceberg"},
		{"mysql", "migrates_to", "postgresql"},
		{"cve:CVE-2026-0001", "affects", "kubernetes"},
		{"dagster", "alternative_to", "apache-airflow"},
	} {
		f := facts[want]
		if f == nil {
			t.Errorf("missing %v", want)
			continue
		}
		if len(f.evidence) != 1 || f.evidence[0].ID != d.id || f.evidence[0].Sentence == "" {
			t.Errorf("%v: evidence %+v", want, f.evidence)
		}
	}
}

// The graph links every entity to its class and its maker (shown when it
// makes two of them, or is named), keeps a typed fact rather than a mere
// co-mention of the same two, and ranks and groups the whole of it.
func TestKnowledgeJoinsTheGraph(t *testing.T) {
	var docs []*gdoc
	for i := range 3 {
		docs = append(docs, &gdoc{id: uuid.New(), title: "dbt and Snowflake", published: time.Now().Add(-time.Duration(i) * time.Hour),
			body: "dbt Labs releases dbt Fusion. dbt integrates with Snowflake and BigQuery."})
	}
	g := Graph{Nodes: []GNode{}, Edges: []GEdge{}}
	addKnowledge(&g, docs, kgContext{corpus: knowledge.NewCorpus(termsOf(docs)), notes: map[string][]GItem{}, weeks: 1, minNPMI: 0.15})
	has := map[string]bool{}
	for _, e := range g.Edges {
		has[e.From+" "+e.Predicate+" "+e.To] = true
	}
	for _, want := range []string{
		"e:dbt is_a k:Data transformation",
		"e:dbt developed_by e:dbt-labs",
		"e:dbt integrates_with e:snowflake",
		"e:dbt-labs releases e:dbt",
	} {
		if !has[want] {
			t.Errorf("missing %q in %v", want, has)
		}
	}
	if has["e:dbt mentioned_with e:snowflake"] || has["e:bigquery developed_by e:google"] {
		t.Errorf("a co-mention beside a fact, or a maker of one: %v", has)
	}
	if g.Stats.Entities == 0 || len(g.Communities) == 0 {
		t.Errorf("stats %+v, communities %v", g.Stats, g.Communities)
	}
	for _, n := range g.Nodes {
		if n.Rank < 0 || n.Rank > 1 {
			t.Errorf("%s rank %v", n.ID, n.Rank)
		}
	}
}

// What the owner's real news got wrong once: a product named like a tool,
// an organization "released" by another, a feed's footer, a link's host,
// and a broad concept taken as a stated fact.
func TestFactsLeaveOutWhatTheNewsDoNotState(t *testing.T) {
	d := &gdoc{id: uuid.New(), title: "News", published: time.Now(), body: "Meta ships Muse Spark Open, its open-weight model. " +
		"OpenAI releases Intelligent UI and Anthropic a new model. " +
		"The post Parquet vs Avro vs ORC appeared first on Dremio. " +
		"Try it [here](https://docs.cloud.google.com/bigquery/docs) and tell Google. " +
		"Keep AI agents away from regulated data using native Snowflake controls."}
	_, _, facts, _ := kgRead([]*gdoc{d})
	for t3 := range facts {
		switch {
		case t3.s == "apache-spark" || t3.o == "apache-spark":
			t.Errorf("Muse Spark read as Apache Spark: %v", t3)
		case t3.p == "releases" && t3.o == "anthropic":
			t.Errorf("an organization released: %v", t3)
		case t3.s == "dremio" || t3.o == "dremio":
			t.Errorf("a footer read as a fact: %v", t3)
		case t3.s == "bigquery" || t3.o == "bigquery":
			t.Errorf("a link read as a mention: %v", t3)
		case t3.p != "mentioned_with" && (entityType(t3.s) == "concept" || entityType(t3.o) == "concept"):
			t.Errorf("a concept in a stated fact: %v", t3)
		}
	}
}

// A fact counts the articles that state it, not its sentences, and a fork
// the ontology already knows gets one edge however often the news says it.
func TestFactsCountArticlesAndKnownForksOnce(t *testing.T) {
	one := &gdoc{id: uuid.New(), title: "Warehouses", published: time.Now(),
		body: "Snowflake and Databricks met. Snowflake and Databricks talked again."}
	_, _, facts, _ := kgRead([]*gdoc{one})
	for tr, f := range facts {
		if f.n != 1 {
			t.Errorf("%v counted %d times in one article", tr, f.n)
		}
	}
	var docs []*gdoc
	for i := range 2 {
		docs = append(docs, &gdoc{id: uuid.New(), title: "Valkey", published: time.Now().Add(-time.Duration(i) * time.Hour),
			body: "Valkey, a fork of Redis, ships a new release. Valkey is a fork of Redis."})
	}
	g := Graph{Nodes: []GNode{}, Edges: []GEdge{}}
	addKnowledge(&g, docs, kgContext{corpus: knowledge.NewCorpus(termsOf(docs)), notes: map[string][]GItem{}, weeks: 1, minNPMI: 0.15})
	seen := map[string]int{}
	for _, e := range g.Edges {
		if seen[e.From+e.Predicate+e.To]++; seen[e.From+e.Predicate+e.To] > 1 {
			t.Errorf("edge %s %s %s twice", e.From, e.Predicate, e.To)
		}
	}
}
