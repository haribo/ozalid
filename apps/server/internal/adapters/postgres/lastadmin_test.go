package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
)

// admins is how many accounts can still administer the instance.
func admins(t *testing.T, ctx context.Context, repo *postgres.Repository) int {
	t.Helper()
	var n int
	if err := repo.Pool().QueryRow(ctx,
		"SELECT count(*) FROM users WHERE is_admin AND deactivated_at IS NULL").Scan(&n); err != nil {
		t.Fatalf("counting the administrators: %v", err)
	}
	return n
}

// anAdmin makes one, and takes it away again when the test is over.
func anAdmin(t *testing.T, ctx context.Context, repo *postgres.Repository, name string) account.Account {
	t.Helper()
	made, err := repo.CreateAccount(ctx, name, anAddress(t), true)
	if err != nil {
		t.Fatalf("creating an administrator: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", made.ID) })
	return made
}

// only leaves exactly one administrator standing — the one it returns — by
// demoting every other. The instance is shared with the rest of this package,
// so "the last administrator" has to be arranged rather than assumed.
func only(t *testing.T, ctx context.Context, repo *postgres.Repository) account.Account {
	t.Helper()
	standing := anAdmin(t, ctx, repo, "the last one")

	var others []string
	rows, err := repo.Pool().Query(ctx,
		"SELECT id FROM users WHERE is_admin AND deactivated_at IS NULL AND id <> $1", standing.ID)
	if err != nil {
		t.Fatalf("reading the other administrators: %v", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scanning: %v", err)
		}
		others = append(others, id)
	}
	rows.Close()

	for _, id := range others {
		if _, err := repo.Pool().Exec(ctx,
			"UPDATE users SET is_admin = false WHERE id = $1", id); err != nil {
			t.Fatalf("standing the others down: %v", err)
		}
		t.Cleanup(func() {
			_, _ = repo.Pool().Exec(ctx, "UPDATE users SET is_admin = true WHERE id = $1", id)
		})
	}
	if got := admins(t, ctx, repo); got != 1 {
		t.Fatalf("%d administrators standing, want exactly 1 for this test to mean anything", got)
	}
	return standing
}

func TestTheLastAdministratorCannotDeactivateThemselves(t *testing.T) {
	ctx, repo := accountFixture(t)
	last := only(t, ctx, repo)

	svc := account.New(repo, repo)
	if err := svc.Deactivate(ctx, last.ID); !errors.Is(err, account.ErrLastAdmin) {
		t.Errorf("Deactivate = %v, want ErrLastAdmin", err)
	}

	// And they are still standing, so the refusal is not cosmetic.
	if got := admins(t, ctx, repo); got != 1 {
		t.Errorf("%d administrators left, want the refusal to have changed nothing", got)
	}
}

func TestTheLastAdministratorCannotDemoteThemselves(t *testing.T) {
	ctx, repo := accountFixture(t)
	last := only(t, ctx, repo)

	svc := account.New(repo, repo)
	if err := svc.SetAdmin(ctx, last.ID, false); !errors.Is(err, account.ErrLastAdmin) {
		t.Errorf("SetAdmin(false) = %v, want ErrLastAdmin", err)
	}
	if got := admins(t, ctx, repo); got != 1 {
		t.Errorf("%d administrators left, want 1", got)
	}
}

func TestWithTwoAdministratorsEitherMayGo(t *testing.T) {
	ctx, repo := accountFixture(t)
	last := only(t, ctx, repo)
	second := anAdmin(t, ctx, repo, "the second")

	svc := account.New(repo, repo)
	if err := svc.Deactivate(ctx, second.ID); err != nil {
		t.Fatalf("deactivating one of two: %v", err)
	}
	// And now the remaining one cannot: the guard counts, it does not remember.
	if err := svc.Deactivate(ctx, last.ID); !errors.Is(err, account.ErrLastAdmin) {
		t.Errorf("deactivating the remaining one = %v, want ErrLastAdmin", err)
	}
}

func TestARoleSetWronglyAtCreationIsNotPermanent(t *testing.T) {
	ctx, repo := accountFixture(t)
	only(t, ctx, repo)

	made, err := repo.CreateAccount(ctx, "meant to administer", anAddress(t), false)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", made.ID) })

	svc := account.New(repo, repo)
	if err := svc.SetAdmin(ctx, made.ID, true); err != nil {
		t.Fatalf("promoting: %v", err)
	}
	if got := admins(t, ctx, repo); got != 2 {
		t.Fatalf("%d administrators after promoting, want 2", got)
	}
	if err := svc.SetAdmin(ctx, made.ID, false); err != nil {
		t.Fatalf("demoting: %v", err)
	}
	if got := admins(t, ctx, repo); got != 1 {
		t.Errorf("%d administrators after demoting, want 1", got)
	}
}

func TestTwoAdministratorsRemovingEachOtherAtOnceLeaveOneStanding(t *testing.T) {
	ctx, repo := accountFixture(t)
	first := only(t, ctx, repo)
	second := anAdmin(t, ctx, repo, "the other one")

	// The failure this guards is a read followed by a write: each request sees
	// the other still standing and both succeed. The refusal is in the WHERE
	// clause, so the second write matches no row instead.
	svc := account.New(repo, repo)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, id := range []string{first.ID, second.ID} {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			errs[i] = svc.Deactivate(ctx, id)
		}(i, id)
	}
	wg.Wait()

	if got := admins(t, ctx, repo); got < 1 {
		t.Fatalf("%d administrators left: the instance locked itself out (errors: %v)", got, errs)
	}
	// Exactly one refusal, so one of the two really was stopped rather than both
	// racing through.
	refused := 0
	for _, err := range errs {
		if errors.Is(err, account.ErrLastAdmin) {
			refused++
		}
	}
	if refused != 1 {
		t.Errorf("%d of the two were refused, want exactly 1 (errors: %v)", refused, errs)
	}
}
