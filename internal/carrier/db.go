// Package carrier provides E.164-prefix to SMS-gateway lookups.
// The seed database is hardcoded below; run go generate to refresh from live sources.
//
//go:generate go run ../../cmd/carrier-update/main.go -out generated_db.go
package carrier

import (
	"fmt"
	"strings"
)

// CarrierRecord holds routing metadata for a mobile carrier.
type CarrierRecord struct {
	Prefix       string
	CarrierName  string
	Country      string
	EmailGateway string
}

// prefixDB is the active lookup map. init() seeds it from seedPrefixDB;
// generated_db.go (when present) merges generatedPrefixDB on top.
var prefixDB map[string]CarrierRecord

func init() {
	prefixDB = make(map[string]CarrierRecord, len(seedPrefixDB))
	for k, v := range seedPrefixDB {
		prefixDB[k] = v
	}
}

// seedPrefixDB is the hardcoded fallback. Run go generate to replace with live data.
var seedPrefixDB = map[string]CarrierRecord{
	// United States — AT&T
	"1201": {Prefix: "1201", CarrierName: "AT&T", Country: "US", EmailGateway: "txt.att.net"},
	"1202": {Prefix: "1202", CarrierName: "AT&T", Country: "US", EmailGateway: "txt.att.net"},
	"1203": {Prefix: "1203", CarrierName: "AT&T", Country: "US", EmailGateway: "txt.att.net"},
	// United States — T-Mobile
	"1206": {Prefix: "1206", CarrierName: "T-Mobile", Country: "US", EmailGateway: "tmomail.net"},
	"1212": {Prefix: "1212", CarrierName: "T-Mobile", Country: "US", EmailGateway: "tmomail.net"},
	"1213": {Prefix: "1213", CarrierName: "T-Mobile", Country: "US", EmailGateway: "tmomail.net"},
	// United States — Verizon
	"1302": {Prefix: "1302", CarrierName: "Verizon", Country: "US", EmailGateway: "vtext.com"},
	"1303": {Prefix: "1303", CarrierName: "Verizon", Country: "US", EmailGateway: "vtext.com"},
	"1304": {Prefix: "1304", CarrierName: "Verizon", Country: "US", EmailGateway: "vtext.com"},

	// United Kingdom — Vodafone
	"4477": {Prefix: "4477", CarrierName: "Vodafone", Country: "UK", EmailGateway: "vodafone.net"},
	"4475": {Prefix: "4475", CarrierName: "Vodafone", Country: "UK", EmailGateway: "vodafone.net"},
	// United Kingdom — O2
	"4478": {Prefix: "4478", CarrierName: "O2", Country: "UK", EmailGateway: "o2.co.uk"},
	"4479": {Prefix: "4479", CarrierName: "O2", Country: "UK", EmailGateway: "o2.co.uk"},
	// United Kingdom — EE
	"4474": {Prefix: "4474", CarrierName: "EE", Country: "UK", EmailGateway: "mms.ee.co.uk"},
	"4473": {Prefix: "4473", CarrierName: "EE", Country: "UK", EmailGateway: "mms.ee.co.uk"},

	// India — Airtel
	"9198": {Prefix: "9198", CarrierName: "Airtel", Country: "IN", EmailGateway: "airtelmail.in"},
	"9199": {Prefix: "9199", CarrierName: "Airtel", Country: "IN", EmailGateway: "airtelmail.in"},
	// India — Jio
	"9196": {Prefix: "9196", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	"9197": {Prefix: "9197", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	// India — Vi (Vodafone Idea)
	"9195": {Prefix: "9195", CarrierName: "Vi", Country: "IN", EmailGateway: "vimail.in"},
	"9194": {Prefix: "9194", CarrierName: "Vi", Country: "IN", EmailGateway: "vimail.in"},
}

// Lookup returns the CarrierRecord for e164Number using longest-prefix match
// over 7 down to 4 digits. e164Number may include a leading '+'.
func Lookup(e164Number string) (*CarrierRecord, error) {
	digits := strings.TrimPrefix(e164Number, "+")
	if len(digits) < 4 {
		return nil, fmt.Errorf("carrier: number too short: %q", e164Number)
	}

	maxLen := 7
	if len(digits) < maxLen {
		maxLen = len(digits)
	}
	for l := maxLen; l >= 4; l-- {
		if rec, ok := prefixDB[digits[:l]]; ok {
			return &rec, nil
		}
	}
	return nil, fmt.Errorf("carrier: no record for %q", e164Number)
}

// GatewayAddress returns an SMS-to-email address for e164Number in the form
// <digits>@<gateway>, ready to use as an email To: header value.
func GatewayAddress(e164Number string) (string, error) {
	rec, err := Lookup(e164Number)
	if err != nil {
		return "", err
	}
	digits := strings.TrimPrefix(e164Number, "+")
	return digits + "@" + rec.EmailGateway, nil
}
