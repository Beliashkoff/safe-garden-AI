package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// SMTPConfig configures the net/smtp-based Mailer.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string // empty → no AUTH (dev / MailHog)
	Password string
	From     string // envelope + From header address
	FromName string
	UseTLS   bool // true → implicit TLS on connect (Yandex 360 :465)
}

// smtpMailer sends mail through an SMTP server. It supports two modes:
//   - implicit TLS (UseTLS): dial a TLS socket directly (port 465, Yandex 360).
//   - plaintext (dev): dial a plain socket (MailHog :1025), no AUTH.
//
// When a username is set, AUTH LOGIN is used. We use a custom loginAuth rather
// than smtp.PlainAuth because PlainAuth refuses to send credentials unless the
// net/smtp client itself negotiated TLS — but with implicit TLS we hand it an
// already-encrypted connection, so that guard would wrongly trip.
type smtpMailer struct {
	cfg  SMTPConfig
	from mail.Address
	now  func() time.Time
}

func newSMTPMailer(cfg SMTPConfig) *smtpMailer {
	return &smtpMailer{
		cfg:  cfg,
		from: mail.Address{Name: cfg.FromName, Address: cfg.From},
		now:  time.Now,
	}
}

func (m *smtpMailer) SendOTP(ctx context.Context, to, code, locale string) error {
	text, html, err := renderOTP(locale, code)
	if err != nil {
		return err
	}
	msg, err := buildMIME(m.from, to, subjectFor(locale), text, html, m.now())
	if err != nil {
		return err
	}
	return m.send(ctx, to, msg)
}

func (m *smtpMailer) send(ctx context.Context, to string, msg []byte) error {
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))

	conn, err := m.dial(ctx, addr)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: smtp client: %w", err)
	}
	defer func() { _ = client.Close() }()

	if m.cfg.Username != "" {
		if err := client.Auth(&loginAuth{username: m.cfg.Username, password: m.cfg.Password}); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}
	if err := client.Mail(m.cfg.From); err != nil {
		return fmt.Errorf("mailer: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailer: DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("mailer: write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: close body: %w", err)
	}
	return client.Quit()
}

// dial honours ctx for the connection setup. For implicit TLS we perform the
// handshake over the ctx-bound raw connection so a hung server cannot block
// past the request deadline.
func (m *smtpMailer) dial(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{}
	raw, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("mailer: dial: %w", err)
	}
	if !m.cfg.UseTLS {
		return raw, nil
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = raw.SetDeadline(deadline)
	}
	tlsConn := tls.Client(raw, &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12})
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = raw.Close()
		return nil, fmt.Errorf("mailer: tls handshake: %w", err)
	}
	_ = tlsConn.SetDeadline(time.Time{}) // clear; per-op deadlines handled by SMTP exchange
	return tlsConn, nil
}

// loginAuth implements the SMTP AUTH LOGIN mechanism. Safe to use here because
// it is only ever invoked over an already-encrypted connection (implicit TLS).
type loginAuth struct {
	username, password string
}

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(fromServer))) {
	case "username:":
		return []byte(a.username), nil
	case "password:":
		return []byte(a.password), nil
	default:
		return nil, fmt.Errorf("mailer: unexpected AUTH LOGIN challenge %q", fromServer)
	}
}
