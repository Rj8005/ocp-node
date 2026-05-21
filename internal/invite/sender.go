// Package invite sends SMS invite links via email-to-SMS gateways.
package invite

import (
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"

	"github.com/Rj8005/ocp-node/internal/carrier"
)

// InviteConfig holds SMTP credentials and invite-link settings.
type InviteConfig struct {
	SMTPHost      string
	SMTPPort      int
	SMTPUsername  string
	SMTPPassword  string
	FromAddress   string
	InviteBaseURL string // e.g. https://opencall.net/invite
}

// GenerateInviteToken returns an 8-char lowercase hex token derived from
// ocpAddress and the current nanosecond timestamp.
func GenerateInviteToken(ocpAddress string) string {
	h := sha256.New()
	h.Write([]byte(ocpAddress))
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))[:8]
}

// SendSMSInvite sends a short invite message to toE164 via its carrier's
// email-to-SMS gateway. fromOCPAddress is embedded in the invite token.
// Returns the matched carrier name on success.
//
// The message body is intentionally kept under 160 characters so it fits
// in a single SMS segment on every carrier.
func SendSMSInvite(cfg InviteConfig, toE164 string, fromOCPAddress string) (string, error) {
	if cfg.SMTPHost == "" {
		return "", fmt.Errorf("invite: SMTP not configured (set SMTP_HOST)")
	}

	rec, err := carrier.Lookup(toE164)
	if err != nil {
		return "", fmt.Errorf("invite: %w", err)
	}

	gatewayAddr, err := carrier.GatewayAddress(toE164)
	if err != nil {
		return "", fmt.Errorf("invite: gateway address: %w", err)
	}

	log.Printf("[invite] carrier=%s gateway=%s to=%s", rec.CarrierName, rec.EmailGateway, gatewayAddr)

	token := GenerateInviteToken(fromOCPAddress)
	base := strings.TrimRight(cfg.InviteBaseURL, "/")
	body := "Call me free on OpenCall: " + base + "/" + token

	// Minimal RFC 2822 envelope — no Subject so carriers pass the body straight
	// through as the SMS text without prepending a subject line.
	msg := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: \r\n\r\n%s\r\n",
		gatewayAddr, cfg.FromAddress, body)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	// Port 465 = implicit TLS (SMTPS).  All other ports go through smtp.SendMail
	// which negotiates STARTTLS automatically when the server offers it.
	if cfg.SMTPPort == 465 {
		if err := sendImplicitTLS(cfg, addr, gatewayAddr, []byte(msg)); err != nil {
			return "", err
		}
		return rec.CarrierName, nil
	}

	var auth smtp.Auth
	if cfg.SMTPUsername != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
	}
	if err := smtp.SendMail(addr, auth, cfg.FromAddress, []string{gatewayAddr}, []byte(msg)); err != nil {
		return "", err
	}
	return rec.CarrierName, nil
}

// sendImplicitTLS handles SMTPS (port 465) where TLS wraps the whole
// connection before the SMTP handshake begins.
func sendImplicitTLS(cfg InviteConfig, addr, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.SMTPHost})
	if err != nil {
		return fmt.Errorf("invite: tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("invite: smtp client: %w", err)
	}
	defer client.Close()

	if cfg.SMTPUsername != "" {
		auth := smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("invite: smtp auth: %w", err)
		}
	}
	if err := client.Mail(cfg.FromAddress); err != nil {
		return fmt.Errorf("invite: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("invite: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("invite: DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("invite: write: %w", err)
	}
	return w.Close()
}
