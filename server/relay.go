package server

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Rj8005/ocp-node/dht"
)

// RelayRecord is the payload a GSM relay node publishes into the DHT so that
// callers can find a relay capable of bridging an internet call to the PSTN.
//
// relay_mode controls what the relay will do:
//   "call"  → relay will only make outbound GSM calls
//   "sms"   → relay will only send SMS messages
//   "both"  → relay can do either (default)
type RelayRecord struct {
	OCPAddress   string    `json:"ocp_address"`
	PhoneNumber  string    `json:"phone_number"`
	RelayMode    string    `json:"relay_mode"`
	CountryCode  string    `json:"country_code"`
	RegisteredAt time.Time `json:"registered_at"`
}

const (
	relayKeyPrefix = "relay:"
	relayTTL       = 24 * time.Hour
)

// storeRelayRecord JSON-encodes rec and writes it to the DHT store under
// "relay:<ocp_address>". Defaults relay_mode to "both" if empty.
func storeRelayRecord(store *dht.Store, rec RelayRecord) error {
	if rec.RelayMode == "" {
		rec.RelayMode = "both"
	}
	rec.RegisteredAt = time.Now()
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	store.Store(relayKeyPrefix+rec.OCPAddress, data, relayTTL)
	log.Printf("[relay] registered ocp=%s mode=%s phone=%s country=%s",
		rec.OCPAddress, rec.RelayMode, rec.PhoneNumber, rec.CountryCode)
	return nil
}

// lookupRelayRecord fetches and decodes a RelayRecord from the DHT store.
// Returns nil, false when the key is absent or expired.
func lookupRelayRecord(store *dht.Store, ocpAddress string) (*RelayRecord, bool) {
	data, ok := store.FindValue(relayKeyPrefix + ocpAddress)
	if !ok {
		return nil, false
	}
	var rec RelayRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		log.Printf("[relay] corrupt record for %s: %v", ocpAddress, err)
		return nil, false
	}
	return &rec, true
}

// HandleRelayRegister handles POST /relay/register
//
// Request body (JSON):
//
//	{
//	  "ocp_address":  "alice@ocp",
//	  "phone_number": "+254712345678",
//	  "relay_mode":   "both",       // optional, defaults to "both"
//	  "country_code": "+254"        // optional
//	}
func (s *HTTPServer) HandleRelayRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var rec RelayRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}
	if rec.OCPAddress == "" || rec.PhoneNumber == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "ocp_address and phone_number are required"})
		return
	}
	if rec.RelayMode != "" && rec.RelayMode != "call" && rec.RelayMode != "sms" && rec.RelayMode != "both" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": `relay_mode must be "call", "sms", or "both"`})
		return
	}

	if err := storeRelayRecord(s.node.Store(), rec); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "registered",
		"ocp":        rec.OCPAddress,
		"relay_mode": rec.RelayMode,
	})
}

// HandleRelayLookup handles GET /relay/lookup?ocp=<ocp_address>
//
// Returns the full RelayRecord including relay_mode, or 404 if not found.
func (s *HTTPServer) HandleRelayLookup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ocp := r.URL.Query().Get("ocp")
	if ocp == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "ocp param required"})
		return
	}

	rec, found := lookupRelayRecord(s.node.Store(), ocp)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "relay not found"})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ocp_address":   rec.OCPAddress,
		"phone_number":  rec.PhoneNumber,
		"relay_mode":    rec.RelayMode,
		"country_code":  rec.CountryCode,
		"registered_at": rec.RegisteredAt,
	})
}
