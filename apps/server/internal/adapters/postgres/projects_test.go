package postgres_test

import (
	"context"
	"testing"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/app/account"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/actor"
	"github.com/haribo/ozalid/apps/server/internal/domain/catalogue"
)

// slugs is what a caller can name out of what they were given.
func slugs(t *testing.T, got []catalogue.Project) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, p := range got {
		out[p.Slug] = true
	}
	return out
}

func TestAMemberSeesTheirProjectsAndNobodyElses(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	other := neighbour(t, ctx, repo)

	person, err := repo.CreateAccount(ctx, "member of one", anAddress(t), false)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", person.ID) })
	if granted, err := repo.Grant(ctx, project.Slug, person.ID, access.Member); err != nil || !granted {
		t.Fatalf("granting: %v, %v", granted, err)
	}

	// Not an administrator: the flag is false, and the neighbouring project is
	// simply not in the answer.
	got, err := repo.ProjectsFor(ctx, actor.Actor{ID: person.ID, Kind: actor.Human}, false)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	seen := slugs(t, got)
	if !seen[project.Slug] {
		t.Error("a member does not see the project they belong to")
	}
	if seen[other.Slug] {
		t.Error("a member sees a project they do not belong to")
	}
}

func TestAnAdministratorSeesEveryProjectsName(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	other := neighbour(t, ctx, repo)

	admin, err := repo.CreateAccount(ctx, "runs the instance", anAddress(t), true)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", admin.ID) })

	// A member of nothing, and they see both — the name, never what is inside
	// (product.md §8.2).
	got, err := repo.ProjectsFor(ctx, actor.Actor{ID: admin.ID, Kind: actor.Human}, true)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	seen := slugs(t, got)
	if !seen[project.Slug] || !seen[other.Slug] {
		t.Errorf("an administrator sees %v, want both %q and %q", seen, project.Slug, other.Slug)
	}
}

func TestAPersonWhoBelongsToNothingSeesNothing(t *testing.T) {
	ctx, repo, _, _ := intakeFixture(t)

	nobody, err := repo.CreateAccount(ctx, "just hired", anAddress(t), false)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", nobody.ID) })

	// An empty list, not an error: belonging to nothing is what a new account
	// is, and it is a legitimate state.
	got, err := repo.ProjectsFor(ctx, actor.Actor{ID: nobody.ID, Kind: actor.Human}, false)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("a member of nothing sees %d projects, want none", len(got))
	}
}

func TestAProgramSeesItsOneProjectEvenWhenToldItAdministers(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)
	neighbour(t, ctx, repo)

	owner, err := repo.CreateAccount(ctx, "owner", anAddress(t), true)
	if err != nil {
		t.Fatalf("creating the owner: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", owner.ID) })
	bot, err := repo.CreateServiceAccount(ctx, project.Slug, "the runner", owner.ID, access.Member)
	if err != nil {
		t.Fatalf("creating the service account: %v", err)
	}

	// The flag is true and it changes nothing: a service account belongs to one
	// project and administers nothing (ADR 0018).
	got, err := repo.ProjectsFor(ctx, actor.Actor{ID: bot.ID, Kind: actor.Machine}, true)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	if len(got) != 1 || got[0].Slug != project.Slug {
		t.Errorf("a program sees %v, want only %q", slugs(t, got), project.Slug)
	}
}

func TestADeactivatedAccountLeavesTheAccessList(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)

	person, err := repo.CreateAccount(ctx, "leaving the team", anAddress(t), false)
	if err != nil {
		t.Fatalf("creating the account: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", person.ID) })
	if _, err := repo.Grant(ctx, project.Slug, person.ID, access.Member); err != nil {
		t.Fatalf("granting: %v", err)
	}

	listed, err := repo.Members(ctx, project.Slug)
	if err != nil {
		t.Fatalf("listing the members: %v", err)
	}
	if !holds(listed, person.ID) {
		t.Fatal("the member is not listed before deactivation, so this proves nothing")
	}

	if _, err := repo.DeactivateAccount(ctx, person.ID); err != nil {
		t.Fatalf("deactivating: %v", err)
	}

	// This page answers "who reaches this project". A deactivated account
	// reaches nothing, so it has no line — it stays on /accounts, which answers
	// a different question (product.md §8.2).
	after, err := repo.Members(ctx, project.Slug)
	if err != nil {
		t.Fatalf("listing again: %v", err)
	}
	if holds(after, person.ID) {
		t.Error("a deactivated account is still listed among who reaches the project")
	}

	// And the membership row survives, so this is a filter and not a deletion.
	var rows int
	if err := repo.Pool().QueryRow(ctx,
		"SELECT count(*) FROM project_members WHERE user_id = $1", person.ID).Scan(&rows); err != nil {
		t.Fatalf("counting the memberships: %v", err)
	}
	if rows != 1 {
		t.Errorf("the membership row count is %d, want it untouched at 1", rows)
	}
}

func TestAProgramWithNoTokenIsListedAndSaysSo(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)

	owner, err := repo.CreateAccount(ctx, "owner", anAddress(t), true)
	if err != nil {
		t.Fatalf("creating the owner: %v", err)
	}
	t.Cleanup(func() { _, _ = repo.Pool().Exec(ctx, "DELETE FROM users WHERE id = $1", owner.ID) })
	bot, err := repo.CreateServiceAccount(ctx, project.Slug, "keyless", owner.ID, access.Member)
	if err != nil {
		t.Fatalf("creating the service account: %v", err)
	}
	minted, err := repo.MintToken(ctx, project.Slug, bot.ID, "the only one")
	if err != nil {
		t.Fatalf("minting: %v", err)
	}

	if got := tokensOf(t, repo, ctx, project.Slug, bot.ID); got == nil || *got != 1 {
		t.Fatalf("tokens = %v, want 1", got)
	}

	if err := repo.RetireToken(ctx, project.Slug, bot.ID, minted.ID); err != nil {
		t.Fatalf("retiring: %v", err)
	}

	// Nothing guards the last token, so this state is reachable. The program is
	// alive and cannot authenticate — a missing key, which somebody can fix,
	// unlike a retired account.
	got := tokensOf(t, repo, ctx, project.Slug, bot.ID)
	if got == nil {
		t.Fatal("a program with no token vanished from the access list")
	}
	if *got != 0 {
		t.Errorf("tokens = %d, want 0", *got)
	}
}

func TestAPersonCarriesNoTokenCount(t *testing.T) {
	ctx, repo, project, _ := intakeFixture(t)

	listed, err := repo.Members(ctx, project.Slug)
	if err != nil {
		t.Fatalf("listing: %v", err)
	}
	for _, m := range listed {
		if m.IsPerson && m.Tokens != nil {
			t.Errorf("%s is a person and carries a token count of %d", m.Name, *m.Tokens)
		}
		if !m.IsPerson && m.Tokens == nil {
			t.Errorf("%s is a program and carries no token count", m.Name)
		}
	}
}

func holds(members []account.Membership, id string) bool {
	for _, m := range members {
		if m.AccountID == id {
			return true
		}
	}
	return false
}

func tokensOf(t *testing.T, repo *postgres.Repository, ctx context.Context, slug, id string) *int {
	t.Helper()
	listed, err := repo.Members(ctx, slug)
	if err != nil {
		t.Fatalf("listing the members: %v", err)
	}
	for _, m := range listed {
		if m.AccountID == id {
			return m.Tokens
		}
	}
	return nil
}
