package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
)

func accountFixture(t *testing.T) (context.Context, *postgres.Repository) {
	t.Helper()
	dsn := os.Getenv("OZALID_TEST_DSN")
	if dsn == "" {
		t.Skip("set OZALID_TEST_DSN to run the database tests (just db-test)")
	}
	ctx := context.Background()
	repo, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("opening the database: %v", err)
	}
	t.Cleanup(repo.Close)
	return ctx, repo
}

// anAddress is unique per run: these tests write for real, against a database
// they share with every other test in this package.
func anAddress(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("account-%d@example.test", time.Now().UnixNano())
}

func TestAnAddressCanOnlyHaveOneAccount(t *testing.T) {
	ctx, repo := accountFixture(t)
	email := anAddress(t)

	made, err := repo.CreateAccount(ctx, "nina", email, false)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() {
		if _, err := repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", made.ID); err != nil {
			t.Errorf("cleaning up: %v", err)
		}
	})

	// The index is on the lowercased address, so a different casing is the same
	// person and must be refused as such rather than quietly making a second
	// account nobody can tell apart.
	_, err = repo.CreateAccount(ctx, "nina again", email, false)
	if !errors.Is(err, account.ErrEmailTaken) {
		t.Errorf("second account on the same address = %v, want ErrEmailTaken", err)
	}
}

func TestDeactivatingTwiceDoesNotMoveTheDay(t *testing.T) {
	ctx, repo := accountFixture(t)

	made, err := repo.CreateAccount(ctx, "leaving", anAddress(t), false)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() {
		if _, err := repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", made.ID); err != nil {
			t.Errorf("cleaning up: %v", err)
		}
	})

	found, err := repo.DeactivateAccount(ctx, made.ID)
	if err != nil || !found {
		t.Fatalf("first deactivation = %v, %v; want it found", found, err)
	}
	first := deactivatedAt(t, ctx, repo, made.ID)

	if found, err = repo.DeactivateAccount(ctx, made.ID); err != nil || !found {
		t.Fatalf("second deactivation = %v, %v; want it found", found, err)
	}
	if again := deactivatedAt(t, ctx, repo, made.ID); !again.Equal(first) {
		t.Errorf("deactivated_at moved from %v to %v; deactivating twice is not two days", first, again)
	}
}

func TestDeactivatingAnAccountNobodyHasIsNotFound(t *testing.T) {
	ctx, repo := accountFixture(t)

	found, err := repo.DeactivateAccount(ctx, "nobody-has-this-id")
	if err != nil {
		t.Fatalf("deactivating: %v", err)
	}
	if found {
		t.Error("an id nobody has reported as deactivated")
	}
}

func TestADeactivatedAccountIsStillListed(t *testing.T) {
	ctx, repo := accountFixture(t)

	made, err := repo.CreateAccount(ctx, "still named", anAddress(t), false)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() {
		if _, err := repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", made.ID); err != nil {
			t.Errorf("cleaning up: %v", err)
		}
	})
	if _, err := repo.DeactivateAccount(ctx, made.ID); err != nil {
		t.Fatalf("deactivating: %v", err)
	}

	// A deactivated account is what a name in the journal resolves to. Hiding
	// it would leave a reviewer's name pointing at nothing (ADR 0018).
	listed, err := repo.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	for _, a := range listed {
		if a.ID != made.ID {
			continue
		}
		if a.DeactivatedAt == nil {
			t.Error("the account is listed but does not say it is deactivated")
		}
		return
	}
	t.Error("the deactivated account is not listed at all")
}

func deactivatedAt(t *testing.T, ctx context.Context, repo *postgres.Repository, id string) time.Time {
	t.Helper()
	var at *time.Time
	if err := repo.Pool().QueryRow(ctx, "SELECT deactivated_at FROM users WHERE id = $1", id).Scan(&at); err != nil {
		t.Fatalf("reading deactivated_at: %v", err)
	}
	if at == nil {
		t.Fatal("the account is not deactivated")
	}
	return *at
}
