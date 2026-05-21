// Package server provides HTTP handler functions for invite-related endpoints.
// These are registered in the top-level server package via its Start() mux.
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Rj8005/ocp-node/internal/invite"
)

// apiClient is shared across all outbound API calls.
// A 15-second timeout prevents slow external services from blocking handlers.
var apiClient = &http.Client{Timeout: 15 * time.Second}

// channelResponse is the JSON shape returned by GET /invite/channels.
type channelResponse struct {
	Name     string `json:"name"`
	DeepLink string `json:"deeplink"`
	API      bool   `json:"api"`
	Icon     string `json:"icon"`
}

// iconFor maps a channel name to the icon identifier used by the frontend.
func iconFor(name string) string {
	switch name {
	case "WhatsApp":
		return "whatsapp"
	case "Telegram":
		return "telegram"
	case "Viber":
		return "viber"
	case "LINE":
		return "line"
	case "KakaoTalk":
		return "kakaotalk"
	case "SMS":
		return "sms"
	case "TextBelt":
		return "sms" // renders using the SMS icon
	case "Signal":
		return "signal"
	case "Zalo":
		return "zalo"
	case "WeChat":
		return "wechat"
	default:
		return strings.ToLower(strings.ReplaceAll(name, " ", ""))
	}
}

// HandleGetChannels handles GET /invite/channels
//
// Query params:
//
//	country    ISO 3166-1 alpha-2 code (e.g. "IN"). Falls back to DEFAULT.
//	phone      E.164 number (e.g. "+919800505720"). Required to expand deep links.
//	invite_url Full invite URL (e.g. "https://opencall.net/join/abc123").
//
// Deep links are only expanded when both phone and invite_url are provided;
// otherwise "deeplink" is returned as an empty string.
func HandleGetChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	phone := r.URL.Query().Get("phone")
	inviteURL := r.URL.Query().Get("invite_url")
	if country == "" {
		country = "DEFAULT"
	}

	channels := invite.GetChannels(country)

	out := make([]channelResponse, 0, len(channels))
	for _, ch := range channels {
		dl := ""
		if phone != "" && inviteURL != "" {
			dl = invite.BuildDeepLink(ch, phone, inviteURL)
		}
		out = append(out, channelResponse{
			Name:     ch.Name,
			DeepLink: dl,
			API:      invite.IsAPIChannel(ch),
			Icon:     iconFor(ch.Name),
		})
	}

	log.Printf("[http] invite/channels country=%s n=%d phone=%s", country, len(out), phone)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// HandleSendInvite handles POST /invite/send
//
// Request body:
//
//	{"to": "+919800505720", "channel": "textbelt", "inviteURL": "https://opencall.net/join/abc123"}
//
// Supported channels: "textbelt", "whatsapp_api"
// Response: {"status": "sent", "channel": "<channel>"}
func HandleSendInvite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		To        string `json:"to"`
		Channel   string `json:"channel"`
		InviteURL string `json:"inviteURL"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.To == "" || req.Channel == "" || req.InviteURL == "" {
		http.Error(w, `"to", "channel", and "inviteURL" are required`, http.StatusBadRequest)
		return
	}

	msg := "Call me free on OpenCall: " + req.InviteURL

	var sendErr error
	switch strings.ToLower(req.Channel) {
	case "textbelt":
		sendErr = SendTextBelt(req.To, msg)
	case "whatsapp_api":
		sendErr = SendWhatsAppAPI(req.To, req.InviteURL)
	default:
		http.Error(w, "unsupported channel: "+req.Channel, http.StatusBadRequest)
		return
	}

	if sendErr != nil {
		log.Printf("[http] invite/send error channel=%s to=%s: %v", req.Channel, req.To, sendErr)
		http.Error(w, sendErr.Error(), http.StatusBadGateway)
		return
	}

	log.Printf("[http] invite/send sent channel=%s to=%s", req.Channel, req.To)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "sent",
		"channel": req.Channel,
	})
}

// SendTextBelt sends message to toE164 via the TextBelt SMS API.
// The key "textbelt" is the public free-tier key (1 SMS/day per IP).
// Set the key via the TEXTBELT_KEY env var to use a paid quota.
func SendTextBelt(toE164 string, message string) error {
	key := os.Getenv("TEXTBELT_KEY")
	if key == "" {
		key = "textbelt" // free tier: 1 SMS / day per IP
	}

	resp, err := apiClient.PostForm("https://textbelt.com/text", url.Values{
		"phone":   {toE164},
		"message": {message},
		"key":     {key},
	})
	if err != nil {
		return fmt.Errorf("textbelt: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("textbelt: decode response: %w", err)
	}
	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "send failed"
		}
		return fmt.Errorf("textbelt: %s", errMsg)
	}
	return nil
}

// SendWhatsAppAPI sends an invite via the WhatsApp Cloud API using the
// pre-approved message template named "ocp_invite".
//
// Required env vars:
//
//	WHATSAPP_TOKEN    Bearer token from Meta developer console
//	WHATSAPP_PHONE_ID Numeric phone number ID (not the display number)
//
// The template must have one body parameter: the invite URL.
func SendWhatsAppAPI(toE164 string, inviteURL string) error {
	token := os.Getenv("WHATSAPP_TOKEN")
	phoneID := os.Getenv("WHATSAPP_PHONE_ID")
	if token == "" || phoneID == "" {
		return fmt.Errorf("whatsapp: WHATSAPP_TOKEN and WHATSAPP_PHONE_ID must be set")
	}

	// WhatsApp requires the number without '+'.
	number := strings.TrimPrefix(toE164, "+")

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                number,
		"type":              "template",
		"template": map[string]interface{}{
			"name":     "ocp_invite",
			"language": map[string]string{"code": "en_US"},
			"components": []map[string]interface{}{
				{
					"type": "body",
					"parameters": []map[string]interface{}{
						{"type": "text", "text": inviteURL},
					},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("whatsapp: marshal: %w", err)
	}

	apiURL := "https://graph.facebook.com/v18.0/" + url.PathEscape(phoneID) + "/messages"
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("whatsapp: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := apiClient.Do(req)
	if err != nil {
		return fmt.Errorf("whatsapp: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return fmt.Errorf("whatsapp: API error %d: %s",
			errBody.Error.Code, errBody.Error.Message)
	}
	return nil
}
