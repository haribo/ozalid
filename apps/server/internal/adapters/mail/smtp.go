// Package mail sends the one message ozalid sends.
//
// Sign-in is the only reason this exists (ADR 0019). There is no newsletter, no
// notification, no digest — a second message would be a product decision, not a
// use of this package.
package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Sender delivers a sign-in link.
type Sender interface {
	SendSignInLink(ctx context.Context, to, link string) error
}

// Config is what an operator supplies.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

// Complete reports whether enough was supplied to send anything.
//
// Host and From are the two that cannot be guessed. Credentials are optional: a
// relay on a private network often wants none, and demanding them would rule
// out the simplest deployment there is.
func (c Config) Complete() bool {
	return c.Host != "" && c.Port != "" && c.From != ""
}

// SMTP sends through a relay.
type SMTP struct {
	cfg  Config
	base string
}

// NewSMTP returns a sender. base is the address the instance is reached at,
// which is what the link in the message has to point back to.
func NewSMTP(cfg Config, base string) *SMTP {
	return &SMTP{cfg: cfg, base: strings.TrimSuffix(base, "/")}
}

// SendSignInLink delivers the one message.
func (s *SMTP) SendSignInLink(_ context.Context, to, link string) error {
	url := fmt.Sprintf("%s/sign-in/%s", s.base, link)

	// Plain text, deliberately. A sign-in message has one sentence and one
	// link; HTML would add a way for it to render wrong and nothing else.
	body := strings.Join([]string{
		"From: " + s.cfg.From,
		"To: " + to,
		"Subject: Sign in to ozalid",
		"Content-Type: text/plain; charset=utf-8",
		"",
		"Follow this link to sign in. It works once, and stops working in fifteen minutes.",
		"",
		url,
		"",
		"If you did not ask for it, nothing has happened and you can ignore this.",
		"",
	}, "\r\n")

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}
	addr := s.cfg.Host + ":" + s.cfg.Port
	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(body)); err != nil {
		return fmt.Errorf("sending the sign-in link: %w", err)
	}
	return nil
}
