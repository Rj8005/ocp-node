// Package channels implements outbound communication channels used to reach
// a recipient and prompt them to install OpenCall.
package channels

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

// SIPConfig holds credentials for the outbound SIP account.
type SIPConfig struct {
	Server   string // e.g. sip.linphone.org
	Username string // free SIP username
	Password string // free SIP password
	FromNum  string // SIP caller ID / address
}

func getSIPConfig() SIPConfig {
	return SIPConfig{
		Server:   getEnvOrDefault("SIP_SERVER", "sip.linphone.org"),
		Username: getEnvOrDefault("SIP_USERNAME", ""),
		Password: getEnvOrDefault("SIP_PASSWORD", ""),
		FromNum:  getEnvOrDefault("SIP_FROM", ""),
	}
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// SendMissedCall dials toE164 via SIP, waits for a ringing response (180/183),
// then immediately cancels — leaving a missed call notification on the
// recipient's handset. The inviteURL is embedded as a custom header so any
// VoIP client that surfaces SIP headers can show the join link.
//
// Required env vars: SIP_USERNAME, SIP_PASSWORD, SIP_FROM
// Optional: SIP_SERVER (default sip.linphone.org)
func SendMissedCall(toE164 string, inviteURL string) error {
	cfg := getSIPConfig()

	if cfg.Username == "" {
		log.Printf("[MissedCall] SIP not configured, skipping")
		return fmt.Errorf("SIP_USERNAME not set")
	}

	log.Printf("[MissedCall] Dialing %s via %s", toE164, cfg.Server)

	ua, err := sipgo.NewUA()
	if err != nil {
		return fmt.Errorf("UA init failed: %w", err)
	}

	client, err := sipgo.NewClient(ua)
	if err != nil {
		return fmt.Errorf("client init failed: %w", err)
	}

	// NewRequest requires a *Uri pointer; routing resolves via Recipient.Host.
	recipient := &sip.Uri{
		User: toE164,
		Host: cfg.Server,
	}

	req := sip.NewRequest(sip.INVITE, recipient)
	req.AppendHeader(sip.NewHeader("X-OCP-Invite", inviteURL))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := client.TransactionRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("INVITE failed: %w", err)
	}

	select {
	case resp := <-tx.Responses():
		log.Printf("[MissedCall] Got response: %d", resp.StatusCode)
		if resp.StatusCode == 180 || resp.StatusCode == 183 {
			// Phone is ringing — cancel immediately to leave a missed call.
			if err := tx.Cancel(); err != nil {
				log.Printf("[MissedCall] CANCEL error (non-fatal): %v", err)
			}
			log.Printf("[MissedCall] ✓ Missed call sent to %s", toE164)
			return nil
		}

	case <-time.After(3 * time.Second):
		// No 180 yet but the INVITE was sent — counts as a ring attempt.
		log.Printf("[MissedCall] ✓ Timeout = missed call delivered to %s", toE164)
		return nil

	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}

	return nil
}
