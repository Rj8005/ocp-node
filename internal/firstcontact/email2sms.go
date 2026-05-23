package firstcontact

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
)

// ── Current working email-to-SMS gateways (2025) ──────────
// AT&T shut down June 2025
// T-Mobile shut down Nov/Dec 2024
// Verizon phasing out March 2027 (intermittent)

var GatewayDB = map[string][]string{

	// ══ UNITED STATES (still working) ══
	"1": {}, // No reliable US gateways — use TextBelt

	// Specific US carriers still working
	"1_boost":   {"sms.myboostmobile.com"},
	"1_fi":      {"msg.fi.google.com"},
	"1_metro":   {"mymetropcs.com"},
	"1_cricket": {}, // AT&T network — dead
	"1_verizon": {"vtext.com"}, // until March 2027

	// ══ CANADA ══
	"1_ca_rogers":  {"pcs.rogers.com"},
	"1_ca_telus":   {"msg.telus.com"},
	"1_ca_bell":    {"txt.bell.ca"},
	"1_ca_freedom": {"txt.freedommobile.ca"},
	"1_ca_virgin":  {"vmobile.ca"},
	"1_ca_koodo":   {"msg.telus.com"},

	// ══ UNITED KINGDOM ══
	"44_vodafone": {"vodafone.net"},
	"44_o2":       {"o2.co.uk"},
	"44_ee":       {"mms.ee.co.uk"},
	"44_three":    {"three.co.uk"},
	"44_giff":     {"giffgaff.com"},
	"44_virgin":   {"vmbtext.co.uk"},

	// ══ INDIA ══
	"91_airtel": {"airtelmail.in"},
	"91_jio":    {"jio.com"},
	"91_vi":     {"vimail.in"},
	"91_bsnl":   {"bsnl.in"},

	// ══ AUSTRALIA ══
	"61_telstra":  {"sms.telstra.com"},
	"61_optus":    {"optusmobile.com.au"},
	"61_vodafone": {"vodafone.com.au"},
	"61_tpg":      {"tpg.com.au"},

	// ══ GERMANY ══
	"49_telekom":  {"t-mobile-sms.de"},
	"49_vodafone": {"vodafone-sms.de"},
	"49_o2":       {"o2online.de"},
	"49_eplus":    {"eplus.de"},

	// ══ FRANCE ══
	"33_orange":   {"orange.fr"},
	"33_sfr":      {"sfr.fr"},
	"33_bouygues": {"bouyguestelecom.fr"},
	"33_free":     {"free.fr"},

	// ══ SPAIN ══
	"34_movistar": {"movistar.net"},
	"34_vodafone": {"vodafone.es"},
	"34_orange":   {"orange.es"},

	// ══ ITALY ══
	"39_tim":     {"tim.it"},
	"39_vodafone": {"vodafone.it"},
	"39_wind":    {"windtre.it"},

	// ══ NETHERLANDS ══
	"31_tmobile": {"t-mobile.nl"},
	"31_kpn":     {"kpn.com"},

	// ══ SWEDEN ══
	"46_telia": {"telia.com"},
	"46_tre":   {"tre.se"},

	// ══ NORWAY ══
	"47_telenor": {"telenor.com"},
	"47_telia":   {"telia.no"},

	// ══ RUSSIA ══
	"7_mts":     {"mts.ru"},
	"7_beeline": {"beeline.ru"},
	"7_megafon": {"megafon.ru"},
	"7_tele2":   {"tele2.ru"},

	// ══ UKRAINE ══
	"380_kyivstar": {"kyivstar.net"},
	"380_life":     {"life.com.ua"},
	"380_mts":      {"mts.com.ua"},

	// ══ BRAZIL ══
	"55_claro": {"claro.com.br"},
	"55_vivo":  {"vivo.com.br"},
	"55_tim":   {"tim.com.br"},
	"55_oi":    {"oi.com.br"},

	// ══ MEXICO ══
	"52_telcel":   {"telcel.com"},
	"52_movistar": {"movistar.com.mx"},
	"52_att":      {"att.com.mx"},

	// ══ SOUTH AFRICA ══
	"27_vodacom": {"vodacom.co.za"},
	"27_mtn":     {"mtn.co.za"},
	"27_cell":    {"cellc.co.za"},
	"27_telkom":  {"telkomsa.net"},

	// ══ NIGERIA ══
	"234_mtn":    {"mtn.com.ng"},
	"234_airtel": {"airtel.com.ng"},
	"234_glo":    {"gloworld.com"},

	// ══ KENYA ══
	"254_safaricom": {"safaricom.co.ke"},
	"254_airtel":    {"airtel.co.ke"},

	// ══ JAPAN ══
	"81_docomo":   {"docomo.ne.jp"},
	"81_softbank": {"softbank.ne.jp"},
	"81_au":       {"ezweb.ne.jp"},

	// ══ SOUTH KOREA ══
	"82_skt": {"skt.com"},
	"82_kt":  {"kt.com"},
	"82_lgu": {"lgu.co.kr"},

	// ══ SINGAPORE ══
	"65_singtel": {"singtel.com"},
	"65_starhub": {"starhub.com"},
	"65_m1":      {"m1.com.sg"},

	// ══ MALAYSIA ══
	"60_maxis":  {"maxis.com.my"},
	"60_celcom": {"celcom.com.my"},
	"60_digi":   {"digi.com.my"},

	// ══ PHILIPPINES ══
	"63_globe": {"globe.com.ph"},
	"63_smart": {"smart.com.ph"},
	"63_sun":   {"suncellular.com.ph"},

	// ══ NEW ZEALAND ══
	"64_spark":    {"spark.co.nz"},
	"64_vodafone": {"vodafone.co.nz"},
	"64_2degrees": {"2degrees.nz"},

	// ══ PAKISTAN ══
	"92_jazz":    {"jazz.com.pk"},
	"92_telenor": {"telenor.com.pk"},
	"92_zong":    {"zong.com.pk"},
	"92_ufone":   {"ufone.com"},

	// ══ BANGLADESH ══
	"880_grameenphone": {"grameenphone.com"},
	"880_robi":         {"robi.com.bd"},
	"880_banglalink":   {"banglalink.net"},

	// ══ SRI LANKA ══
	"94_mobitel": {"mobitel.lk"},
	"94_dialog":  {"dialog.lk"},
	"94_airtel":  {"airtel.lk"},

	// ══ UAE ══
	"971_etisalat": {"etisalat.ae"},
	"971_du":       {"du.ae"},

	// ══ SAUDI ARABIA ══
	"966_stc":    {"stc.com.sa"},
	"966_mobily": {"mobily.com.sa"},
	"966_zain":   {"zain.com.sa"},

	// ══ EGYPT ══
	"20_orange":   {"orange.com.eg"},
	"20_etisalat": {"etisalat.eg"},
	"20_vodafone": {"vodafone.com.eg"},

	// ══ TURKEY ══
	"90_turkcell": {"turkcell.com.tr"},
	"90_vodafone": {"vodafone.com.tr"},
	"90_turk":     {"turktelekom.com.tr"},

	// ══ IRAN ══
	"98_mci":      {"mci.ir"},
	"98_irancell": {"irancell.ir"},
	"98_rightel":  {"rightel.ir"},

	// ══ INDONESIA ══
	"62_telkomsel": {"telkomsel.com"},
	"62_indosat":   {"indosatooredoo.com"},
	"62_xl":        {"xl.co.id"},
	"62_three":     {"tri.co.id"},

	// ══ VIETNAM ══
	"84_viettel":   {"viettel.vn"},
	"84_vinaphone": {"vinaphone.vn"},
	"84_mobifone":  {"mobifone.vn"},

	// ══ THAILAND ══
	"66_ais":  {"ais.th"},
	"66_dtac": {"dtac.co.th"},
	"66_true": {"truemove-h.com"},

	// ══ GHANA ══
	"233_mtn":      {"mtn.com.gh"},
	"233_vodafone": {"vodafone.com.gh"},
	"233_airtel":   {"airtel.com.gh"},

	// ══ TANZANIA ══
	"255_vodacom": {"vodacom.co.tz"},
	"255_airtel":  {"airtel.co.tz"},
	"255_tigo":    {"tigo.co.tz"},
}

// Prefix to country+carrier mapping
var PrefixToCarrier = map[string]string{
	// Canada (before US to match longer prefix)
	"1416": "1_ca_rogers", "1647": "1_ca_rogers",
	"1604": "1_ca_telus", "1250": "1_ca_telus",
	"1613": "1_ca_bell", "1905": "1_ca_bell",
	"1780": "1_ca_bell", "1514": "1_ca_bell",
	"1403": "1_ca_bell", "1902": "1_ca_bell",

	// UK
	"4477": "44_vodafone", "4475": "44_vodafone",
	"4478": "44_o2", "4479": "44_o2",
	"4474": "44_ee", "4473": "44_ee",
	"4476": "44_three",

	// India — format: country code (91) + first 4 digits of local number
	// Vi/Vodafone-Idea 80xx
	"918000": "91_vi", "918001": "91_vi", "918002": "91_vi",
	"918003": "91_vi", "918004": "91_vi", "918005": "91_vi",
	"918006": "91_vi", "918007": "91_vi", "918008": "91_vi",
	"918009": "91_vi",
	// Vi legacy prefixes
	"919971": "91_vi", "919319": "91_vi",
	"918800": "91_vi", "918287": "91_vi",
	// Airtel 98xxx
	"919810": "91_airtel", "919811": "91_airtel", "919812": "91_airtel",
	"919820": "91_airtel", "919821": "91_airtel", "919876": "91_airtel",
	// Airtel legacy prefixes
	"919198": "91_airtel", "919873": "91_airtel",
	"919871": "91_airtel", "919953": "91_airtel",
	// Jio 89xxx / 93xxx
	"918888": "91_jio", "918889": "91_jio",
	"919321": "91_jio", "919322": "91_jio",
	"918976": "91_jio", "918977": "91_jio",
	// Jio legacy prefixes
	"919188": "91_jio", "919196": "91_jio",
	"919197": "91_jio", "919082": "91_jio",
	"919081": "91_jio", "919080": "91_jio",
	// BSNL
	"919418": "91_bsnl", "919419": "91_bsnl", "919796": "91_bsnl",

	// Australia
	"6140": "61_telstra", "6141": "61_telstra",
	"6142": "61_optus", "6143": "61_optus",
	"6144": "61_vodafone",

	// Germany
	"4915": "49_telekom", "4916": "49_telekom",
	"4917": "49_vodafone", "4918": "49_o2",

	// Brazil
	"5511": "55_claro", "5521": "55_claro",
	"5512": "55_vivo", "5522": "55_vivo",
	"5513": "55_tim", "5523": "55_tim",
	"5514": "55_oi",

	// Russia
	"7916": "7_mts", "7917": "7_mts",
	"7903": "7_beeline", "7905": "7_beeline",
	"7920": "7_megafon", "7921": "7_megafon",
	"7902": "7_tele2", "7908": "7_tele2",

	// Japan
	"8190": "81_docomo", "8180": "81_docomo",
	"8170": "81_softbank", "8090": "81_au",

	// South Africa
	"2782": "27_vodacom", "2783": "27_vodacom",
	"2781": "27_mtn", "2784": "27_cell",

	// Nigeria
	"2348": "234_glo", "2347": "234_airtel",

	// Indonesia
	"6281": "62_telkomsel", "6282": "62_telkomsel",
	"6285": "62_indosat", "6286": "62_xl",
	"6289": "62_three",

	// Pakistan
	"923": "92_jazz", "924": "92_telenor",
	"925": "92_ufone", "926": "92_zong",

	// Philippines
	"639": "63_globe", "6391": "63_smart",
	"6392": "63_sun",

	// Vietnam
	"8496": "84_viettel", "8497": "84_viettel",
	"8498": "84_vinaphone", "8479": "84_mobifone",

	// UAE
	"97150": "971_etisalat", "97155": "971_etisalat",
	"97154": "971_du", "97156": "971_du",

	// Saudi
	"96650": "966_stc", "96653": "966_stc",
	"96655": "966_mobily", "96658": "966_zain",
}

// Countries where email-to-SMS is unreliable — fall back to TextBelt
var TextBeltFallbackCountries = map[string]bool{
	"1":  true, // US — major carriers shut down
	"86": true, // China — no email-to-SMS
	"82": true, // Korea — unreliable
	"66": true, // Thailand — unreliable
	"98": true, // Iran — blocked
	// "91" removed — India uses carrier email-to-SMS gateways directly
}

type SMSResult struct {
	Success bool   `json:"success"`
	Method  string `json:"method"`
	Gateway string `json:"gateway,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SendSMSWithFallback tries email-to-SMS first, falls back to TextBelt.
func SendSMSWithFallback(toE164, message string) SMSResult {
	country := extractCountryCode(toE164)

	// India: try carrier email-to-SMS gateways directly; no TextBelt fallback
	if country == "91" {
		_, gw := findGateway(toE164)
		if gw != "" {
			log.Printf("[SMS] India carrier gateway: %s via %s", toE164, gw)
			err := sendEmailToSMSGateway(toE164, gw, message)
			if err == nil {
				return SMSResult{Success: true, Method: "email_to_sms", Gateway: gw}
			}
			log.Printf("[SMS] India gateway %s failed: %v", gw, err)
		}
		return SMSResult{
			Success: false,
			Method:  "email_to_sms",
			Error:   "India carrier gateway failed — SMTP may not be configured",
		}
	}

	if TextBeltFallbackCountries[country] {
		log.Printf("[SMS] Country %s → TextBelt directly", country)
		return sendViaTextBelt(toE164, message)
	}

	carrier, gw := findGateway(toE164)
	if gw != "" {
		log.Printf("[SMS] Trying email-to-SMS via %s", gw)
		err := sendEmailToSMSGateway(toE164, gw, message)
		if err == nil {
			return SMSResult{Success: true, Method: "email_to_sms", Gateway: gw}
		}
		log.Printf("[SMS] Email-to-SMS failed (%s): %v", gw, err)
	} else {
		log.Printf("[SMS] No gateway found for %s (%s)", toE164, carrier)
	}

	log.Printf("[SMS] Falling back to TextBelt for %s", toE164)
	return sendViaTextBelt(toE164, message)
}

func findGateway(e164 string) (string, string) {
	digits := extractDigits(e164)

	for _, l := range []int{5, 4, 3} {
		if len(digits) >= l {
			prefix := digits[:l]
			if carrier, ok := PrefixToCarrier[prefix]; ok {
				if gateways := GatewayDB[carrier]; len(gateways) > 0 {
					return carrier, gateways[0]
				}
			}
		}
	}

	country := extractCountryCode(e164)
	key := country + "_generic"
	if gateways, ok := GatewayDB[key]; ok && len(gateways) > 0 {
		return country, gateways[0]
	}

	return "", ""
}

func sendEmailToSMSGateway(toE164, gateway, message string) error {
	local := getLocalNumber(toE164)
	toEmail := local + "@" + gateway

	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPass := os.Getenv("SMTP_PASS")
	smtpFrom := os.Getenv("SMTP_FROM")

	if smtpHost == "" {
		smtpHost = "smtp.gmail.com"
	}
	if smtpPort == "" {
		smtpPort = "587"
	}
	if smtpFrom == "" {
		smtpFrom = smtpUser
	}
	if smtpUser == "" {
		return fmt.Errorf("SMTP not configured")
	}

	if len(message) > 140 {
		message = message[:137] + "..."
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: \r\n\r\n%s",
		smtpFrom, toEmail, message)

	addr := smtpHost + ":" + smtpPort
	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)
	_ = &tls.Config{ServerName: smtpHost} // kept for future TLS dial upgrade

	return smtp.SendMail(addr, auth, smtpFrom, []string{toEmail}, []byte(msg))
}

func sendViaTextBelt(toE164, message string) SMSResult {
	resp, err := http.PostForm("https://textbelt.com/text", url.Values{
		"phone":   {toE164},
		"message": {message},
		"key":     {"textbelt"},
	})
	if err != nil {
		return SMSResult{Success: false, Method: "textbelt", Error: err.Error()}
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if success, _ := result["success"].(bool); success {
		return SMSResult{Success: true, Method: "textbelt"}
	}
	errMsg, _ := result["error"].(string)
	return SMSResult{Success: false, Method: "textbelt", Error: errMsg}
}

func extractDigits(s string) string {
	out := ""
	for _, c := range s {
		if c >= '0' && c <= '9' {
			out += string(c)
		}
	}
	return out
}

func extractCountryCode(e164 string) string {
	digits := extractDigits(e164)
	prefixes := map[string]string{
		"880": "880", "971": "971", "966": "966",
		"255": "255", "254": "254", "234": "234",
		"233": "233", "380": "380", "886": "886",
		"598": "598", "44": "44", "49": "49",
		"33": "33", "34": "34", "39": "39",
		"31": "31", "46": "46", "47": "47",
		"91": "91", "86": "86", "81": "81",
		"82": "82", "66": "66", "62": "62",
		"63": "63", "84": "84", "92": "92",
		"98": "98", "90": "90", "20": "20",
		"27": "27", "55": "55", "52": "52",
		"61": "61", "64": "64", "65": "65",
		"60": "60", "94": "94", "7": "7",
		"1": "1",
	}
	for _, l := range []int{3, 2, 1} {
		if len(digits) >= l {
			if cc, ok := prefixes[digits[:l]]; ok {
				return cc
			}
		}
	}
	return ""
}

func getLocalNumber(e164 string) string {
	digits := extractDigits(e164)
	cc := extractCountryCode(e164)
	if len(digits) > len(cc) {
		return digits[len(cc):]
	}
	return digits
}
