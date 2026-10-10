package app

import (
	"context"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/thoutmose/axon/api/internal/graph"
)

// This file sorts last, so its tests run after every other test in the package.

// Operations covered by another package's tests (internal/server).
var notExercisedHere = []string{"health", "ready"}

func TestZZEveryOperationIsExercised(t *testing.T) {
	loadSpec(t)
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			if _, ok := covered.Load(op.OperationID); !ok && !slices.Contains(notExercisedHere, op.OperationID) {
				t.Errorf("no test exercises %s %s (%s)", method, path, op.OperationID)
			}
		}
	}
}

// Every domain table that hangs off entities must be a registered type, so
// export and integrity checks cover it. This lives in the app package because
// only the app imports every domain package.
func TestZZEveryEntityTableIsRegistered(t *testing.T) {
	rows, err := pool.Query(context.Background(), `
		SELECT DISTINCT tc.table_name FROM information_schema.table_constraints tc
		JOIN information_schema.constraint_column_usage cu ON cu.constraint_name = tc.constraint_name
		WHERE tc.constraint_type = 'FOREIGN KEY' AND cu.table_name = 'entities' AND tc.table_name <> 'links'`)
	if err != nil {
		t.Fatal(err)
	}
	tables, _ := pgx.CollectRows(rows, pgx.RowTo[string])
	registered := map[string]bool{}
	for _, typ := range graph.Types() {
		registered[typ.Table] = true
	}
	for _, table := range tables {
		if !registered[table] {
			t.Errorf("table %s references entities but no graph.Type registers it", table)
		}
	}
}
