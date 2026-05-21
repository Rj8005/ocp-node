// cmd/carrier-update downloads carrier prefix data from Ofcom (UK) and NANPA
// (North America) and generates internal/carrier/generated_db.go as a Go map
// literal, so the full database compiles into the binary with no runtime files.
//
// Usage (from module root):
//
//	go run ./cmd/carrier-update                    # default output path
//	go run ./cmd/carrier-update -out /custom/path/generated_db.go
//	go run ./cmd/carrier-update -ofcom-csv https://... -nanpa-csv https://...
//
// Or via go generate from internal/carrier/:
//
//	go generate ./internal/carrier/
package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Default source URLs.
// Ofcom: landing page is scraped for .csv links.
// NANPA: direct CSV from the NANPA CO Code assignment report.
// Override either with the -ofcom-csv / -nanpa-csv flags if the defaults stop working.
const (
	ofcomPageURL = "https://www.ofcom.org.uk/phones-and-broadband/phone-numbers/numbering/number-range-downloads"
	nanpaPageURL = "https://www.nanpa.com/reports/reports_npa_nxx.html"
)

// record is an intermediate row before writing the generated file.
type record struct {
	prefix      string
	carrierName string
	country     string
	gateway     string
}

// gatewayRules maps lowercase carrier-name substrings → (gateway domain, ISO country).
// Rules are checked in declaration order; first match wins. Put more-specific
// strings (e.g. "vodafone idea") before shorter ones (e.g. "vodafone").
var gatewayRules = []struct {
	keyword string
	gateway string
	country string
}{
	// ── United States ────────────────────────────────────────────────────────
	{"new cingular", "txt.att.net", "US"},
	{"cingular", "txt.att.net", "US"},
	{"at&t", "txt.att.net", "US"},
	{"cellco", "vtext.com", "US"}, // Cellco Partnership dba Verizon
	{"verizon", "vtext.com", "US"},
	{"t-mobile", "tmomail.net", "US"},
	{"tmobile", "tmomail.net", "US"},
	{"metro by t-mobile", "tmomail.net", "US"},
	{"metropcs", "mymetropcs.com", "US"},
	{"metro pcs", "mymetropcs.com", "US"},
	{"sprint", "messaging.sprintpcs.com", "US"},
	{"boost", "sms.myboostmobile.com", "US"},
	{"cricket", "sms.cricketwireless.net", "US"},
	{"us cellular", "email.uscc.net", "US"},
	{"uscellular", "email.uscc.net", "US"},
	{"tracfone", "mmst5.tracfone.com", "US"},
	{"consumer cellular", "mailmymobile.net", "US"},
	{"google fi", "msg.fi.google.com", "US"},
	{"republic wireless", "text.republicwireless.com", "US"},
	{"dish wireless", "msg.dish.com", "US"},
	// ── Canada ───────────────────────────────────────────────────────────────
	{"bell mobility", "txt.bell.ca", "CA"},
	{"bell canada", "txt.bell.ca", "CA"},
	{"rogers", "pcs.rogers.com", "CA"},
	{"fido", "fido.ca", "CA"},
	{"freedom mobile", "txt.freedommobile.ca", "CA"},
	{"shaw mobile", "msg.telus.com", "CA"},
	{"koodo", "msg.telus.com", "CA"},
	{"telus", "msg.telus.com", "CA"},
	{"virgin mobile ca", "vmobile.ca", "CA"},
	{"sasktel", "sms.sasktel.com", "CA"},
	{"mts mobility", "text.mts.net", "CA"},
	// ── United Kingdom ───────────────────────────────────────────────────────
	{"telefonica uk", "o2.co.uk", "UK"},   // Telefónica = O2 UK
	{"telefonica", "o2.co.uk", "UK"},      // fallback for any Telefónica entity
	{"hutchison 3g", "3mail.co.uk", "UK"}, // Hutchison = Three UK
	{"hutchison", "3mail.co.uk", "UK"},
	{"everything everywhere", "mms.ee.co.uk", "UK"}, // legacy EE name
	{"ee limited", "mms.ee.co.uk", "UK"},
	{"vodafone limited", "vodafone.net", "UK"},
	{"vodafone", "vodafone.net", "UK"},
	{"o2 uk", "o2.co.uk", "UK"},
	{"o2", "o2.co.uk", "UK"},
	{"ee ", "mms.ee.co.uk", "UK"}, // trailing space avoids false matches
	{"orange personal", "orange.net", "UK"},
	{"orange", "orange.net", "UK"},
	{"three", "3mail.co.uk", "UK"},
	{"british telecommunications", "btmobile.bt.com", "UK"},
	{"bt mobile", "btmobile.bt.com", "UK"},
	{"bt plc", "btmobile.bt.com", "UK"},
	{"virgin mobile", "vxtras.com", "UK"},
	{"tesco mobile", "o2.co.uk", "UK"}, // Tesco uses O2 network
	{"sky mobile", "skynet.co.uk", "UK"},
	{"talktalk", "talktalk.net", "UK"},
	// ── India ────────────────────────────────────────────────────────────────
	{"vodafone idea", "vimail.in", "IN"}, // merged entity (Vi)
	{"vi limited", "vimail.in", "IN"},
	{"idea cellular", "vimail.in", "IN"},
	{"bharti airtel", "airtelmail.in", "IN"},
	{"airtel", "airtelmail.in", "IN"},
	{"reliance jio", "jio.com", "IN"},
	{"jio", "jio.com", "IN"},
	{"bsnl", "bsnl.in", "IN"},
	{"mtnl", "mtnl.net.in", "IN"},
}

// normalizeToGateway returns the SMS email gateway and ISO country code for a
// carrier name, or empty strings if no rule matches.
func normalizeToGateway(carrierName string) (gateway, country string) {
	lower := strings.ToLower(carrierName)
	for _, rule := range gatewayRules {
		if strings.Contains(lower, rule.keyword) {
			return rule.gateway, rule.country
		}
	}
	return "", ""
}

// fetchURL downloads a URL and returns its body bytes.
func fetchURL(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ocp-carrier-updater/1.0; +https://github.com/Rj8005/ocp-node)")
	req.Header.Set("Accept", "text/csv,text/plain,*/*")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, rawURL)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// extractCSVLinks parses an HTML page and returns all fully-resolved URLs
// whose href contains ".csv" (case-insensitive).
func extractCSVLinks(htmlBody []byte, pageURL string) []string {
	base, err := url.Parse(pageURL)
	if err != nil {
		return nil
	}
	content := string(htmlBody)
	var links []string
	seen := map[string]bool{}
	pos := 0
	lower := strings.ToLower(content)
	for pos < len(lower) {
		i := strings.Index(lower[pos:], "href=")
		if i < 0 {
			break
		}
		pos += i + 5
		if pos >= len(content) {
			break
		}
		q := content[pos] // opening quote char
		if q != '"' && q != '\'' {
			continue
		}
		pos++
		j := strings.IndexByte(content[pos:], q)
		if j < 0 {
			break
		}
		link := content[pos : pos+j]
		pos += j + 1
		if !strings.Contains(strings.ToLower(link), ".csv") {
			continue
		}
		ref, err := url.Parse(link)
		if err != nil {
			continue
		}
		resolved := base.ResolveReference(ref).String()
		if !seen[resolved] {
			seen[resolved] = true
			links = append(links, resolved)
		}
	}
	return links
}

// digitsOnly returns s with all non-digit characters removed.
func digitsOnly(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// ukToPrefix converts a UK national number range string (e.g. "07700" or
// "07700900000") to an E.164 prefix of 4–6 digits (e.g. "447700").
// Returns "" for non-mobile/non-geographic ranges or invalid input.
func ukToPrefix(raw string) string {
	digits := digitsOnly(raw)
	// Need at least "0" + 4 national digits to form a 6-char E.164 prefix.
	if len(digits) < 5 || digits[0] != '0' {
		return ""
	}
	// Only mobile (07) and geographic (01, 02) ranges carry SMS gateways.
	second := digits[1]
	if second != '7' && second != '1' && second != '2' {
		return ""
	}
	national := digits[1:]    // strip leading 0
	e164 := "44" + national   // prepend UK country code
	if len(e164) > 6 {
		e164 = e164[:6]
	}
	if len(e164) < 4 {
		return ""
	}
	return e164
}

// nanpToPrefix builds a 7-digit NANP E.164 prefix from NPA and NXX strings
// (e.g. NPA="206", NXX="555" → "1206555").
func nanpToPrefix(npa, nxx string) string {
	npa = digitsOnly(strings.TrimSpace(npa))
	nxx = digitsOnly(strings.TrimSpace(nxx))
	if len(npa) != 3 || len(nxx) != 3 {
		return ""
	}
	return "1" + npa + nxx
}

// findCol returns the index of the first header that contains any of the
// given lowercase substrings, or -1 if none match.
func findCol(headers []string, needles ...string) int {
	for i, h := range headers {
		h = strings.ToLower(strings.TrimSpace(h))
		for _, n := range needles {
			if strings.Contains(h, n) {
				return i
			}
		}
	}
	return -1
}

// parseOfcomCSV parses a CSV from Ofcom's number-range download page.
//
// Expected columns (auto-detected by header keywords):
//   - number/range column: the UK number prefix (e.g. "07700")
//   - holder/operator column: the licensed operator name
//
// Rows for non-mobile ranges or unknown operators are silently skipped.
func parseOfcomCSV(data []byte) ([]record, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1

	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read headers: %w", err)
	}

	numCol := findCol(headers, "number", "range", "no.")
	holderCol := findCol(headers, "holder", "operator", "company", "allocated", "licensee", "grantee")
	if numCol < 0 || holderCol < 0 {
		return nil, fmt.Errorf("cannot identify columns from headers %v", headers)
	}

	var records []record
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // skip malformed rows
		}
		if numCol >= len(row) || holderCol >= len(row) {
			continue
		}
		num := strings.TrimSpace(row[numCol])
		holder := strings.TrimSpace(row[holderCol])
		if num == "" || holder == "" {
			continue
		}
		prefix := ukToPrefix(num)
		if prefix == "" {
			continue
		}
		gw, country := normalizeToGateway(holder)
		if gw == "" {
			continue
		}
		if country == "" {
			country = "UK"
		}
		records = append(records, record{
			prefix:      prefix,
			carrierName: holder,
			country:     country,
			gateway:     gw,
		})
	}
	return records, nil
}

// parseNANPCSV parses a CSV from the NANPA NPA-NXX assignment report.
//
// Expected columns (auto-detected):
//   - NPA: 3-digit area code
//   - NXX / CO Code: 3-digit exchange
//   - Company / Carrier: licensee name
//   - State / Province: for CA/US differentiation (optional)
//
// Rows for unknown carriers are silently skipped.
func parseNANPCSV(data []byte) ([]record, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1

	// NANPA reports sometimes have a title row before the header; skip non-header
	// rows until we find one that looks like a real header.
	var headers []string
	for {
		row, err := r.Read()
		if err == io.EOF {
			return nil, fmt.Errorf("no header row found")
		}
		if err != nil {
			continue
		}
		if len(row) < 2 {
			continue
		}
		// A real header row will contain "npa" or "area"
		joined := strings.ToLower(strings.Join(row, " "))
		if strings.Contains(joined, "npa") || strings.Contains(joined, "area code") {
			headers = row
			break
		}
	}

	npaCol := findCol(headers, "npa", "area code")
	nxxCol := findCol(headers, "nxx", "co code", "exchange", "nxx_x")
	companyCol := findCol(headers, "company", "carrier", "holder", "entity", "licensee")
	stateCol := findCol(headers, "state", "province", "jurisdiction")

	if npaCol < 0 || nxxCol < 0 || companyCol < 0 {
		return nil, fmt.Errorf("cannot identify NPA/NXX/company columns from headers %v", headers)
	}

	maxRequired := npaCol
	if nxxCol > maxRequired {
		maxRequired = nxxCol
	}
	if companyCol > maxRequired {
		maxRequired = companyCol
	}

	var records []record
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(row) <= maxRequired {
			continue
		}
		company := strings.TrimSpace(row[companyCol])
		if company == "" {
			continue
		}
		prefix := nanpToPrefix(row[npaCol], row[nxxCol])
		if prefix == "" {
			continue
		}

		// Distinguish Canadian provinces from US states.
		country := "US"
		if stateCol >= 0 && stateCol < len(row) {
			switch strings.ToUpper(strings.TrimSpace(row[stateCol])) {
			case "AB", "BC", "MB", "NB", "NL", "NS", "NT", "NU", "ON", "PE", "QC", "SK", "YT":
				country = "CA"
			}
		}
		gw, gwCountry := normalizeToGateway(company)
		if gw == "" {
			continue
		}
		if gwCountry != "" {
			country = gwCountry
		}
		records = append(records, record{
			prefix:      prefix,
			carrierName: company,
			country:     country,
			gateway:     gw,
		})
	}
	return records, nil
}

// dedup merges records with the same prefix (last-writer wins) and returns a
// slice sorted by prefix for deterministic generated output.
func dedup(records []record) []record {
	m := make(map[string]record, len(records))
	for _, r := range records {
		m[r.prefix] = r
	}
	out := make([]record, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].prefix < out[j].prefix })
	return out
}

// writeGenerated formats and writes the generated Go source to outPath.
func writeGenerated(records []record, sources []string, outPath string) error {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "// Code generated by cmd/carrier-update; DO NOT EDIT.\n")
	fmt.Fprintf(&buf, "// Run 'go generate ./internal/carrier/' from the module root to refresh.\n")
	fmt.Fprintf(&buf, "// Generated: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&buf, "// Sources:   %s\n", strings.Join(sources, ", "))
	fmt.Fprintf(&buf, "// Records:   %d\n\n", len(records))
	fmt.Fprintf(&buf, "package carrier\n\n")
	fmt.Fprintf(&buf, "func init() {\n")
	fmt.Fprintf(&buf, "\tfor k, v := range generatedPrefixDB {\n")
	fmt.Fprintf(&buf, "\t\tprefixDB[k] = v\n")
	fmt.Fprintf(&buf, "\t}\n")
	fmt.Fprintf(&buf, "}\n\n")
	fmt.Fprintf(&buf, "// generatedPrefixDB contains records downloaded from upstream sources.\n")
	fmt.Fprintf(&buf, "// Entries here override seedPrefixDB entries for the same prefix.\n")
	fmt.Fprintf(&buf, "var generatedPrefixDB = map[string]CarrierRecord{\n")
	for _, r := range records {
		fmt.Fprintf(&buf, "\t%q: {Prefix: %q, CarrierName: %q, Country: %q, EmailGateway: %q},\n",
			r.prefix, r.prefix, r.carrierName, r.country, r.gateway)
	}
	fmt.Fprintf(&buf, "}\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted source so the caller can diagnose the template bug.
		debugPath := outPath + ".debug"
		_ = os.WriteFile(debugPath, buf.Bytes(), 0644)
		return fmt.Errorf("go/format failed: %w (raw source written to %s)", err, debugPath)
	}

	if dir := filepath.Dir(outPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return os.WriteFile(outPath, formatted, 0644)
}

func main() {
	ofcomURL := flag.String("ofcom-url", ofcomPageURL,
		"Ofcom number-range downloads page; scraped for .csv links")
	ofcomCSV := flag.String("ofcom-csv", "",
		"direct Ofcom CSV URL (skips page scraping)")
	nanpaURL := flag.String("nanpa-url", nanpaPageURL,
		"NANPA NPA-NXX reports page; scraped for .csv links")
	nanpaCSV := flag.String("nanpa-csv", "",
		"direct NANPA CSV URL (skips page scraping)")
	out := flag.String("out", "internal/carrier/generated_db.go",
		"output path for generated_db.go")
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("[carrier-update] ")

	var all []record
	var sources []string

	// ── Ofcom UK ────────────────────────────────────────────────────────────
	ofcomFinal := *ofcomCSV
	if ofcomFinal == "" {
		log.Printf("fetching Ofcom page: %s", *ofcomURL)
		body, err := fetchURL(*ofcomURL)
		if err != nil {
			log.Printf("WARNING: Ofcom page fetch failed: %v", err)
		} else {
			links := extractCSVLinks(body, *ofcomURL)
			log.Printf("found %d CSV link(s) on Ofcom page", len(links))
			if len(links) > 0 {
				ofcomFinal = links[0]
				log.Printf("using: %s", ofcomFinal)
			}
		}
	}
	if ofcomFinal != "" {
		log.Printf("downloading Ofcom CSV: %s", ofcomFinal)
		data, err := fetchURL(ofcomFinal)
		if err != nil {
			log.Printf("WARNING: Ofcom CSV download failed: %v", err)
		} else {
			recs, err := parseOfcomCSV(data)
			if err != nil {
				log.Printf("WARNING: Ofcom CSV parse failed: %v", err)
			} else {
				log.Printf("Ofcom: %d UK records parsed", len(recs))
				all = append(all, recs...)
				sources = append(sources, "Ofcom")
			}
		}
	}

	// ── NANPA ───────────────────────────────────────────────────────────────
	nanpaFinal := *nanpaCSV
	if nanpaFinal == "" {
		log.Printf("fetching NANPA page: %s", *nanpaURL)
		body, err := fetchURL(*nanpaURL)
		if err != nil {
			log.Printf("WARNING: NANPA page fetch failed: %v", err)
		} else {
			links := extractCSVLinks(body, *nanpaURL)
			log.Printf("found %d CSV link(s) on NANPA page", len(links))
			if len(links) > 0 {
				nanpaFinal = links[0]
				log.Printf("using: %s", nanpaFinal)
			}
		}
	}
	if nanpaFinal != "" {
		log.Printf("downloading NANPA CSV: %s", nanpaFinal)
		data, err := fetchURL(nanpaFinal)
		if err != nil {
			log.Printf("WARNING: NANPA CSV download failed: %v", err)
		} else {
			recs, err := parseNANPCSV(data)
			if err != nil {
				log.Printf("WARNING: NANPA CSV parse failed: %v", err)
			} else {
				log.Printf("NANPA: %d records parsed", len(recs))
				all = append(all, recs...)
				sources = append(sources, "NANPA")
			}
		}
	}

	if len(all) == 0 {
		log.Fatalf("no records parsed from any source — not overwriting %s\n"+
			"Tip: supply direct CSV URLs with -ofcom-csv and/or -nanpa-csv", *out)
	}

	deduped := dedup(all)
	log.Printf("writing %d unique prefixes → %s", len(deduped), *out)
	if err := writeGenerated(deduped, sources, *out); err != nil {
		log.Fatalf("write failed: %v", err)
	}
	log.Printf("done — verify with: go build ./...")
}
