package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/Rj8005/ocp-node/internal/invite"
)

type USSDCallback struct {
	SessionID   string `json:"sessionId"`
	PhoneNumber string `json:"phoneNumber"`
	Text        string `json:"text"`
	NetworkCode string `json:"networkCode"`
	ServiceCode string `json:"serviceCode"`
}

func HandleUSSDCallback(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	cb := USSDCallback{
		SessionID:   r.FormValue("sessionId"),
		PhoneNumber: r.FormValue("phoneNumber"),
		Text:        r.FormValue("text"),
		NetworkCode: r.FormValue("networkCode"),
		ServiceCode: r.FormValue("serviceCode"),
	}

	log.Printf("[USSD] callback from %s text=%q session=%s",
		cb.PhoneNumber, cb.Text, cb.SessionID)

	choice := strings.TrimSpace(cb.Text)

	switch choice {
	case "1":
		// User accepted — bridge the call
		callID := invite.LookupUSSDSession(cb.SessionID)
		if callID != "" {
			BridgeUSSDCall(cb.PhoneNumber, callID)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("END Connecting your call now. Please wait."))
		} else {
			w.Write([]byte("END Session expired. Please ask caller to try again."))
		}

	case "2":
		// User declined
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("END Call declined."))

	case "3":
		// User wants the link — send via SMS
		callID := invite.LookupUSSDSession(cb.SessionID)
		inviteURL := "https://opencall.net/join/" + callID
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("END Link sent to your phone: " + inviteURL))

	default:
		// First hit — show menu
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("CON OpenCall\nYou have a free incoming call\n\n1. Accept\n2. Decline\n3. Get link via SMS"))
	}
}

func BridgeUSSDCall(calleePhone string, callID string) {
	log.Printf("[USSD] bridging call %s to %s", callID, calleePhone)
	// TODO: signal your WebSocket server to bridge callID to calleePhone
	// This fires a webhook or internal message to your signal server
	BridgeCallByID(callID, calleePhone)
}

type BridgeRequest struct {
	CallID  string `json:"callId"`
	Callee  string `json:"callee"`
	Channel string `json:"channel"`
}

func BridgeCallByID(callID string, calleePhone string) {
	payload, _ := json.Marshal(BridgeRequest{
		CallID:  callID,
		Callee:  calleePhone,
		Channel: "ussd",
	})
	log.Printf("[USSD] bridge payload: %s", payload)
	// Hook into your existing signal server bridge flow here
}
