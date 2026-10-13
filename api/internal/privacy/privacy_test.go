package privacy

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Two packages keeping data under one consent both hear its withdrawal.
func TestEveryListenerHearsAWithdrawal(t *testing.T) {
	var heard []string
	OnWithdraw("test_purpose", func(context.Context, pgx.Tx, uuid.UUID) error { heard = append(heard, "a"); return nil })
	OnWithdraw("test_purpose", func(context.Context, pgx.Tx, uuid.UUID) error { heard = append(heard, "b"); return nil })
	defer delete(onWithdraw, "test_purpose")
	if err := withdrawn(context.Background(), nil, uuid.Nil, "test_purpose"); err != nil || len(heard) != 2 {
		t.Fatalf("heard %v, %v", heard, err)
	}
}
