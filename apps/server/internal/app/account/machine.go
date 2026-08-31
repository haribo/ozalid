package account

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/haribo/ozalid/apps/server/internal/domain/access"
)

// ServiceAccount is a program with an account, as an administrator sees it.
type ServiceAccount struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// MintedToken is a token at the one moment it can be read.
//
// It exists in this shape exactly once, on the way out of the response that
// created it. Only its hash is stored, so nothing can give it back.
type MintedToken struct {
	ID        string
	Label     string
	Token     string
	CreatedAt time.Time
}

// ServiceToken is a token as it can be seen afterwards: what it is for, and
// whether anything still presents it. Never the token itself.
type ServiceToken struct {
	ID    string
	Label string
	// LastUsedAt is nil while nothing has ever presented it. It is what makes a
	// rotation finishable: an operator watches it stop moving on the old token
	// before retiring it.
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

// Machines stores the programs that reach a project, and their credentials.
type Machines interface {
	CreateServiceAccount(ctx context.Context, slug, name, ownerID string, rights access.Rights) (ServiceAccount, error)
	DeactivateServiceAccount(ctx context.Context, slug, id string) (bool, error)
	MintToken(ctx context.Context, slug, serviceAccountID, label string) (MintedToken, error)
	ListTokens(ctx context.Context, slug, serviceAccountID string) ([]ServiceToken, error)
	RetireToken(ctx context.Context, slug, serviceAccountID, tokenID string) error
}

// ErrLabelRequired means a token was minted without saying what it is for.
//
// The friction is intended: a token nobody can name is a token nobody dares
// retire, and an account ends up carrying credentials no one will ever remove.
var ErrLabelRequired = errors.New("account: a token needs a label")

// CreateServiceAccount makes a program an account on one project, with its
// first token.
//
// The account belongs to that project and never moves: one project per service
// account is enforced by a partial unique index rather than trusted (ADR 0018).
func (s *Service) CreateServiceAccount(ctx context.Context, slug, name, ownerID string, rights access.Rights, label string) (ServiceAccount, MintedToken, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ServiceAccount{}, MintedToken{}, ErrNameRequired
	}
	if strings.TrimSpace(label) == "" {
		return ServiceAccount{}, MintedToken{}, ErrLabelRequired
	}
	switch rights {
	case access.Reader, access.Member:
	default:
		return ServiceAccount{}, MintedToken{}, ErrUnknownRights
	}

	made, err := s.machines.CreateServiceAccount(ctx, slug, name, ownerID, rights)
	if err != nil {
		return ServiceAccount{}, MintedToken{}, err
	}
	token, err := s.machines.MintToken(ctx, slug, made.ID, strings.TrimSpace(label))
	if err != nil {
		return ServiceAccount{}, MintedToken{}, err
	}
	return made, token, nil
}

// DeactivateServiceAccount retires a program. Its tokens stop opening anything
// at once, and everything it pushed stays and keeps naming it (ADR 0018).
func (s *Service) DeactivateServiceAccount(ctx context.Context, slug, id string) error {
	found, err := s.machines.DeactivateServiceAccount(ctx, slug, id)
	if err != nil {
		return err
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

// MintToken adds a token to a program. It does not retire the others: that is
// what makes a rotation survivable rather than an outage.
func (s *Service) MintToken(ctx context.Context, slug, serviceAccountID, label string) (MintedToken, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return MintedToken{}, ErrLabelRequired
	}
	return s.machines.MintToken(ctx, slug, serviceAccountID, label)
}

// Tokens returns what a program holds, without the tokens themselves.
func (s *Service) Tokens(ctx context.Context, slug, serviceAccountID string) ([]ServiceToken, error) {
	return s.machines.ListTokens(ctx, slug, serviceAccountID)
}

// RetireToken stops one token working, leaving the others alone.
func (s *Service) RetireToken(ctx context.Context, slug, serviceAccountID, tokenID string) error {
	return s.machines.RetireToken(ctx, slug, serviceAccountID, tokenID)
}
