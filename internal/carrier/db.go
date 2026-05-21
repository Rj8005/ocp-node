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
	Country      string // ISO 3166-1 alpha-2
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
// Keys are E.164 digit prefixes (no leading +), longest-prefix match wins on lookup.
// UK/US/AU/DE/BR/CA entries use 4-digit keys; India uses 6-digit keys (91 + 4-digit
// subscriber prefix) because India mobile allocations require more specificity.
var seedPrefixDB = map[string]CarrierRecord{

	// ── United Kingdom (GB, +44) ──────────────────────────────────────────────
	"4473": {Prefix: "4473", CarrierName: "EE", Country: "GB", EmailGateway: "mms.ee.co.uk"},
	"4474": {Prefix: "4474", CarrierName: "EE", Country: "GB", EmailGateway: "mms.ee.co.uk"},
	"4475": {Prefix: "4475", CarrierName: "Vodafone", Country: "GB", EmailGateway: "vodafone.net"},
	"4476": {Prefix: "4476", CarrierName: "Three", Country: "GB", EmailGateway: "three.co.uk"},
	"4477": {Prefix: "4477", CarrierName: "Vodafone", Country: "GB", EmailGateway: "vodafone.net"},
	"4478": {Prefix: "4478", CarrierName: "O2", Country: "GB", EmailGateway: "o2.co.uk"},
	"4479": {Prefix: "4479", CarrierName: "O2", Country: "GB", EmailGateway: "o2.co.uk"},

	// ── United States (US, +1) ────────────────────────────────────────────────
	"1201": {Prefix: "1201", CarrierName: "AT&T", Country: "US", EmailGateway: "txt.att.net"},
	"1202": {Prefix: "1202", CarrierName: "AT&T", Country: "US", EmailGateway: "txt.att.net"},
	"1404": {Prefix: "1404", CarrierName: "AT&T", Country: "US", EmailGateway: "txt.att.net"},
	"1206": {Prefix: "1206", CarrierName: "T-Mobile", Country: "US", EmailGateway: "tmomail.net"},
	"1212": {Prefix: "1212", CarrierName: "T-Mobile", Country: "US", EmailGateway: "tmomail.net"},
	"1310": {Prefix: "1310", CarrierName: "T-Mobile", Country: "US", EmailGateway: "tmomail.net"},
	"1302": {Prefix: "1302", CarrierName: "Verizon", Country: "US", EmailGateway: "vtext.com"},
	"1303": {Prefix: "1303", CarrierName: "Verizon", Country: "US", EmailGateway: "vtext.com"},
	"1732": {Prefix: "1732", CarrierName: "Verizon", Country: "US", EmailGateway: "vtext.com"},
	"1816": {Prefix: "1816", CarrierName: "Sprint", Country: "US", EmailGateway: "messaging.sprintpcs.com"},
	"1913": {Prefix: "1913", CarrierName: "Sprint", Country: "US", EmailGateway: "messaging.sprintpcs.com"},

	// ── Canada (CA, +1) ───────────────────────────────────────────────────────
	"1416": {Prefix: "1416", CarrierName: "Rogers", Country: "CA", EmailGateway: "pcs.rogers.com"},
	"1647": {Prefix: "1647", CarrierName: "Rogers", Country: "CA", EmailGateway: "pcs.rogers.com"},
	"1613": {Prefix: "1613", CarrierName: "Bell", Country: "CA", EmailGateway: "txt.bell.ca"},
	"1905": {Prefix: "1905", CarrierName: "Bell", Country: "CA", EmailGateway: "txt.bell.ca"},
	"1604": {Prefix: "1604", CarrierName: "Telus", Country: "CA", EmailGateway: "msg.telus.com"},
	"1250": {Prefix: "1250", CarrierName: "Telus", Country: "CA", EmailGateway: "msg.telus.com"},

	// ── India (IN, +91) — 6-digit keys: "91" + 4-digit subscriber prefix ──────
	"919080": {Prefix: "919080", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	"919081": {Prefix: "919081", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	"919082": {Prefix: "919082", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	"919188": {Prefix: "919188", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	"919196": {Prefix: "919196", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	"919197": {Prefix: "919197", CarrierName: "Jio", Country: "IN", EmailGateway: "jio.com"},
	"917042": {Prefix: "917042", CarrierName: "Airtel", Country: "IN", EmailGateway: "airtelmail.in"},
	"919198": {Prefix: "919198", CarrierName: "Airtel", Country: "IN", EmailGateway: "airtelmail.in"},
	"919871": {Prefix: "919871", CarrierName: "Airtel", Country: "IN", EmailGateway: "airtelmail.in"},
	"919873": {Prefix: "919873", CarrierName: "Airtel", Country: "IN", EmailGateway: "airtelmail.in"},
	"919953": {Prefix: "919953", CarrierName: "Airtel", Country: "IN", EmailGateway: "airtelmail.in"},
	"918287": {Prefix: "918287", CarrierName: "Vi", Country: "IN", EmailGateway: "vimail.in"},
	"918800": {Prefix: "918800", CarrierName: "Vi", Country: "IN", EmailGateway: "vimail.in"},
	"919319": {Prefix: "919319", CarrierName: "Vi", Country: "IN", EmailGateway: "vimail.in"},
	"919971": {Prefix: "919971", CarrierName: "Vi", Country: "IN", EmailGateway: "vimail.in"},
	"919418": {Prefix: "919418", CarrierName: "BSNL", Country: "IN", EmailGateway: "bsnl.in"},
	"919419": {Prefix: "919419", CarrierName: "BSNL", Country: "IN", EmailGateway: "bsnl.in"},
	"919796": {Prefix: "919796", CarrierName: "BSNL", Country: "IN", EmailGateway: "bsnl.in"},

	// ── Australia (AU, +61) ───────────────────────────────────────────────────
	"6140": {Prefix: "6140", CarrierName: "Telstra", Country: "AU", EmailGateway: "sms.telstra.com"},
	"6141": {Prefix: "6141", CarrierName: "Telstra", Country: "AU", EmailGateway: "sms.telstra.com"},
	"6142": {Prefix: "6142", CarrierName: "Optus", Country: "AU", EmailGateway: "optusmobile.com.au"},
	"6143": {Prefix: "6143", CarrierName: "Optus", Country: "AU", EmailGateway: "optusmobile.com.au"},
	"6144": {Prefix: "6144", CarrierName: "Vodafone", Country: "AU", EmailGateway: "vodafone.com.au"},

	// ── Germany (DE, +49) ─────────────────────────────────────────────────────
	"4915": {Prefix: "4915", CarrierName: "Telekom", Country: "DE", EmailGateway: "t-mobile-sms.de"},
	"4916": {Prefix: "4916", CarrierName: "Telekom", Country: "DE", EmailGateway: "t-mobile-sms.de"},
	"4917": {Prefix: "4917", CarrierName: "Vodafone", Country: "DE", EmailGateway: "vodafone-sms.de"},
	"4918": {Prefix: "4918", CarrierName: "O2", Country: "DE", EmailGateway: "o2online.de"},

	// ── Brazil (BR, +55) ──────────────────────────────────────────────────────
	"5511": {Prefix: "5511", CarrierName: "Claro", Country: "BR", EmailGateway: "claro.com.br"},
	"5521": {Prefix: "5521", CarrierName: "Claro", Country: "BR", EmailGateway: "claro.com.br"},
	"5512": {Prefix: "5512", CarrierName: "Vivo", Country: "BR", EmailGateway: "vivo.com.br"},
	"5522": {Prefix: "5522", CarrierName: "Vivo", Country: "BR", EmailGateway: "vivo.com.br"},
	"5513": {Prefix: "5513", CarrierName: "TIM", Country: "BR", EmailGateway: "tim.com.br"},
	"5523": {Prefix: "5523", CarrierName: "TIM", Country: "BR", EmailGateway: "tim.com.br"},
}

// countryDB maps E.164 digit prefixes (no leading +) to ISO 3166-1 alpha-2
// country codes. Longer prefixes take precedence; CountryFromE164 tries
// 3-digit then 2-digit then 1-digit.
var countryDB = map[string]string{
	// ── 3-digit prefixes ──────────────────────────────────────────────────────
	"234": "NG", // Nigeria
	"243": "CD", // DR Congo
	"251": "ET", // Ethiopia
	"254": "KE", // Kenya
	"255": "TZ", // Tanzania
	"256": "UG", // Uganda
	"375": "BY", // Belarus
	"380": "UA", // Ukraine
	"880": "BD", // Bangladesh
	// ── 2-digit prefixes ──────────────────────────────────────────────────────
	"20": "EG", // Egypt
	"27": "ZA", // South Africa
	"33": "FR", // France
	"34": "ES", // Spain
	"39": "IT", // Italy
	"44": "GB", // United Kingdom
	"48": "PL", // Poland
	"49": "DE", // Germany
	"52": "MX", // Mexico
	"54": "AR", // Argentina
	"55": "BR", // Brazil
	"57": "CO", // Colombia
	"60": "MY", // Malaysia
	"61": "AU", // Australia
	"62": "ID", // Indonesia
	"63": "PH", // Philippines
	"66": "TH", // Thailand
	"81": "JP", // Japan
	"82": "KR", // South Korea
	"84": "VN", // Vietnam
	"86": "CN", // China
	"90": "TR", // Turkey
	"91": "IN", // India
	"92": "PK", // Pakistan
	"95": "MM", // Myanmar
	"98": "IR", // Iran
	// ── 1-digit prefixes (broadest, checked last) ─────────────────────────────
	"1": "US", // USA / Canada (defaults to US; use carrier lookup to distinguish CA)
	"7": "RU", // Russia / Kazakhstan (defaults to Russia)
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

// CountryFromE164 returns the ISO 3166-1 alpha-2 country code for e164Number
// using longest-prefix match (3 digits → 2 digits → 1 digit).
// Returns an empty string when no match is found.
//
// Notes:
//   - +1 returns "US" (covers Canada too; use carrier Lookup to distinguish).
//   - +7 returns "RU" (covers Kazakhstan too).
func CountryFromE164(e164 string) string {
	digits := strings.TrimPrefix(e164, "+")
	if len(digits) == 0 {
		return ""
	}
	for l := 3; l >= 1; l-- {
		if len(digits) < l {
			continue
		}
		if cc, ok := countryDB[digits[:l]]; ok {
			return cc
		}
	}
	return ""
}
