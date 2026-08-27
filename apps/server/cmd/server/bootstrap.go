package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres"
	"github.com/haribo/ozalid/apps/server/internal/adapters/postgres/sqlcgen"
	"github.com/haribo/ozalid/apps/server/internal/domain/access"
	"github.com/haribo/ozalid/apps/server/internal/domain/credential"
)

// bootstrap opens the door the first time.
//
// To mint a token you have to be an administrator; to be an administrator you
// have to sign in; signing in does not exist yet. Somebody has to go round, and
// this is the way round: a command run once, on the machine, by whoever
// installed it.
//
// It is not an endpoint, so there is nothing to protect and nothing to forget
// to protect. It is not a client either — ADR 0015 forbids a CLI for pushing
// evidence, and this pushes none; it seeds the instance so that clients can
// begin.
//
// The token is printed once. Nothing keeps it: only its hash is stored, so it
// cannot be shown again and a lost one is replaced rather than recovered.
// say writes a line of the report, and stops the command if it cannot.
//
// What this prints is not decoration: the token exists nowhere else, so a write
// that silently failed would leave an operator holding an instance they cannot
// reach.
func say(format string, args ...any) error {
	if _, err := fmt.Fprintf(os.Stdout, format, args...); err != nil {
		return fmt.Errorf("writing the report: %w", err)
	}
	return nil
}

func bootstrap(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	name := flags.String("name", "", "who the first administrator is")
	email := flags.String("email", "", "their address, for signing in once that exists")
	project := flags.String("project", "", "a project to create and put them in, optional")
	account := flags.String("service-account", "", "a service account to create in that project, optional")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *name == "" || *email == "" {
		return fmt.Errorf("bootstrap needs -name and -email")
	}
	if *account != "" && *project == "" {
		return fmt.Errorf("a service account belongs to a project: pass -project too")
	}

	cfg := load()
	store, err := postgres.Open(ctx, cfg.dsn)
	if err != nil {
		return err
	}
	defer store.Close()

	q := store.Queries()
	admin, err := q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Name: *name, Email: *email, IsAdmin: true,
	})
	if err != nil {
		return fmt.Errorf("creating the administrator: %w", err)
	}
	if err := say("administrator %s (%s)\n", admin.Name, admin.ID); err != nil {
		return err
	}

	if *project == "" {
		return nil
	}
	created, err := q.CreateProject(ctx, sqlcgen.CreateProjectParams{
		Slug: *project, Name: *project, IntakePolicy: "per-case",
	})
	if err != nil {
		return fmt.Errorf("creating the project: %w", err)
	}
	// The administrator is put in as a member: administration reaches accounts,
	// never content, so being one grants nothing here (product.md §8.2).
	if err := q.AddProjectMember(ctx, sqlcgen.AddProjectMemberParams{
		ProjectID: created.ID, UserID: &admin.ID, Rights: string(access.Member),
	}); err != nil {
		return fmt.Errorf("adding the administrator to the project: %w", err)
	}
	if err := say("project %s (%s)\n", created.Slug, created.ID); err != nil {
		return err
	}

	if *account == "" {
		return nil
	}
	bot, err := q.CreateServiceAccount(ctx, sqlcgen.CreateServiceAccountParams{
		Name: *account, OwnerID: admin.ID,
	})
	if err != nil {
		return fmt.Errorf("creating the service account: %w", err)
	}
	if err := q.AddProjectMember(ctx, sqlcgen.AddProjectMemberParams{
		ProjectID: created.ID, ServiceAccountID: &bot.ID, Rights: string(access.Member),
	}); err != nil {
		return fmt.Errorf("adding the service account to the project: %w", err)
	}

	token, hash, err := credential.Mint()
	if err != nil {
		return err
	}
	if _, err := q.CreateServiceToken(ctx, sqlcgen.CreateServiceTokenParams{
		ServiceAccountID: bot.ID, Label: "bootstrap", TokenHash: hash,
	}); err != nil {
		return fmt.Errorf("recording the token: %w", err)
	}

	if err := say("service account %s (%s)\n", bot.Name, bot.ID); err != nil {
		return err
	}
	if err := say("\ntoken, shown once and never again:\n\n  %s\n\n", token); err != nil {
		return err
	}
	return nil
}
