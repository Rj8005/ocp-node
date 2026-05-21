// Package server provides HTTP handler functions for invite-related endpoints.
// These are registered in the top-level server package via its Start() mux.
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Rj8005/ocp-node/internal/carrier"
)

// apiClient is shared across all outbound API calls.
// A 15-second timeout prevents slow external services from blocking handlers.
var apiClient = &http.Client{Timeout: 15 * time.Second}

type channelResponse struct {
	Name     string `json:"name"`
	DeepLink string `json:"deeplink"`
	Icon     string `json:"icon"`
}

// HandleGetToken handles GET /invite/token
func HandleGetToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": "abc123",
		"url":   "https://opencall-server.vercel.app/join/abc123",
	})
}

// HandleGetChannels handles GET /invite/channels
//
// Query params:
//
//	country   ISO 3166-1 alpha-2 code (e.g. "IN"). Falls back to default list.
//	number    E.164 number (e.g. "+919800505720"). Used to expand deep links.
func HandleGetChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}

	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	number := strings.TrimPrefix(r.URL.Query().Get("number"), "+")

	var channels []channelResponse
	switch country {
	case "CN":
		channels = []channelResponse{
			{Name: "WeChat", DeepLink: "", Icon: "wechat"},
			{Name: "SMS", DeepLink: "sms:" + number, Icon: "sms"},
		}
	case "JP":
		channels = []channelResponse{
			{Name: "LINE", DeepLink: "https://line.me/ti/p/~" + number, Icon: "line"},
			{Name: "WhatsApp", DeepLink: "https://wa.me/" + number, Icon: "whatsapp"},
			{Name: "SMS", DeepLink: "sms:" + number, Icon: "sms"},
		}
	case "KR":
		channels = []channelResponse{
			{Name: "KakaoTalk", DeepLink: "kakaoplus://friend/" + number, Icon: "kakao"},
			{Name: "WhatsApp", DeepLink: "https://wa.me/" + number, Icon: "whatsapp"},
			{Name: "SMS", DeepLink: "sms:" + number, Icon: "sms"},
		}
	case "RU":
		channels = []channelResponse{
			{Name: "Telegram", DeepLink: "https://t.me/+" + number, Icon: "telegram"},
			{Name: "Viber", DeepLink: "viber://chat?number=%2B" + number, Icon: "viber"},
			{Name: "WhatsApp", DeepLink: "https://wa.me/" + number, Icon: "whatsapp"},
			{Name: "SMS", DeepLink: "sms:" + number, Icon: "sms"},
		}
	default:
		channels = []channelResponse{
			{Name: "WhatsApp", DeepLink: "https://wa.me/" + number, Icon: "whatsapp"},
			{Name: "Telegram", DeepLink: "https://t.me/+" + number, Icon: "telegram"},
			{Name: "SMS", DeepLink: "sms:" + number, Icon: "sms"},
			{Name: "Viber", DeepLink: "viber://chat?number=%2B" + number, Icon: "viber"},
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(channels)
}

// HandleSendInvite handles POST /invite/send
//
// Request body:
//
//	{"to": "+919800505720", "channel": "textbelt", "inviteURL": "https://..."}
//
// Supported channels: "textbelt", "whatsapp_api", "sms_gateway"
func HandleSendInvite(w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")

	switch strings.ToLower(req.Channel) {
	case "textbelt":
		key := os.Getenv("TEXTBELT_KEY")
		if key == "" {
			key = "textbelt"
		}
		resp, err := apiClient.PostForm("https://textbelt.com/text", url.Values{
			"phone":   {req.To},
			"message": {"Join me free on OpenCall: " + req.InviteURL},
			"key":     {key},
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
			return
		}
		defer resp.Body.Close()
		var tbResult map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&tbResult)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":            "ok",
			"channel":           "textbelt",
			"textbelt_response": tbResult,
		})

	case "whatsapp_api":
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  "WhatsApp API token not configured yet",
		})

	case "sms_gateway":
		gatewayAddr, err := carrier.GatewayAddress(req.To)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
			return
		}
		smtpHost := os.Getenv("SMTP_HOST")
		if smtpHost == "" {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": "SMTP not configured (set SMTP_HOST)"})
			return
		}
		smtpPort := os.Getenv("SMTP_PORT")
		if smtpPort == "" {
			smtpPort = "587"
		}
		smtpFrom := os.Getenv("SMTP_FROM")
		smtpUser := os.Getenv("SMTP_USER")
		smtpPass := os.Getenv("SMTP_PASS")
		msg := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: \r\n\r\nJoin me free on OpenCall: %s\r\n",
			gatewayAddr, smtpFrom, req.InviteURL)
		var auth smtp.Auth
		if smtpUser != "" {
			auth = smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
		}
		if err := smtp.SendMail(smtpHost+":"+smtpPort, auth, smtpFrom, []string{gatewayAddr}, []byte(msg)); err != nil {
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"channel": "sms_gateway",
			"gateway": gatewayAddr,
		})

	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"error":  "unsupported channel: " + req.Channel,
		})
	}
}

// HandleTextBeltSend handles POST /reach/textbelt
// Uses the free-tier TextBelt key (1 SMS/day per IP). No configuration needed.
func HandleTextBeltSend(w http.ResponseWriter, r *http.Request) {
	var body struct {
		To        string `json:"to"`
		InviteURL string `json:"inviteURL"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	message := "Call me free on OpenCall — tap to answer, no app needed: " + body.InviteURL

	resp, err := apiClient.PostForm("https://textbelt.com/text", url.Values{
		"phone":   {body.To},
		"message": {message},
		"key":     {"textbelt"},
	})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false, "error": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SendTextBelt sends message to toE164 via the TextBelt SMS API.
// The key "textbelt" is the public free-tier key (1 SMS/day per IP).
// Set the key via the TEXTBELT_KEY env var to use a paid quota.
func SendTextBelt(toE164 string, message string) error {
	key := os.Getenv("TEXTBELT_KEY")
	if key == "" {
		key = "textbelt"
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
func SendWhatsAppAPI(toE164 string, inviteURL string) error {
	token := os.Getenv("WHATSAPP_TOKEN")
	phoneID := os.Getenv("WHATSAPP_PHONE_ID")
	if token == "" || phoneID == "" {
		return fmt.Errorf("whatsapp: WHATSAPP_TOKEN and WHATSAPP_PHONE_ID must be set")
	}

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
