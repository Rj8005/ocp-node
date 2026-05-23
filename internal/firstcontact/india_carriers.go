package firstcontact

// IndiaCarrierDB maps first 4 digits of local number
// (after stripping +91) to carrier.
// Source: DoT National Numbering Plan allocations.
// Note: MNP means ported numbers may differ — this is original allocation.

var IndiaCarrierDB = map[string]string{

	// ══ AIRTEL ══════════════════════════════════════
	// 6xxx series
	"6200": "airtel", "6201": "airtel", "6202": "airtel",
	"6203": "airtel", "6204": "airtel", "6205": "airtel",
	"6206": "airtel", "6207": "airtel", "6208": "airtel",
	"6209": "airtel",
	"6360": "airtel", "6361": "airtel", "6362": "airtel",
	"6363": "airtel", "6364": "airtel", "6365": "airtel",
	"6366": "airtel", "6367": "airtel", "6368": "airtel",
	"6369": "airtel",
	// 7xxx series
	"7003": "airtel", "7004": "airtel", "7005": "airtel",
	"7006": "airtel",
	"7029": "airtel", "7030": "airtel", "7031": "airtel",
	"7032": "airtel", "7033": "airtel", "7034": "airtel",
	"7035": "airtel", "7036": "airtel", "7037": "airtel",
	"7038": "airtel", "7039": "airtel",
	"7055": "airtel", "7056": "airtel", "7057": "airtel",
	"7058": "airtel", "7059": "airtel",
	"7067": "airtel", "7068": "airtel", "7069": "airtel",
	"7073": "airtel", "7074": "airtel", "7075": "airtel",
	"7076": "airtel", "7077": "airtel", "7078": "airtel",
	"7079": "airtel",
	// 8xxx series
	"8000": "airtel", "8001": "airtel", "8002": "airtel",
	"8003": "airtel", "8004": "airtel", "8005": "airtel",
	"8006": "airtel", "8007": "airtel", "8008": "airtel",
	"8009": "airtel",
	"8010": "airtel", "8011": "airtel", "8012": "airtel",
	"8013": "airtel", "8014": "airtel", "8015": "airtel",
	"8016": "airtel", "8017": "airtel", "8018": "airtel",
	"8019": "airtel",
	"8050": "airtel", "8051": "airtel", "8052": "airtel",
	"8053": "airtel", "8054": "airtel", "8055": "airtel",
	"8056": "airtel", "8057": "airtel", "8058": "airtel",
	"8059": "airtel",
	"8095": "airtel", "8096": "airtel", "8097": "airtel",
	"8098": "airtel", "8099": "airtel",
	"8100": "airtel", "8101": "airtel", "8102": "airtel",
	"8103": "airtel", "8105": "airtel", // 8104 → jio
	"8106": "airtel", "8107": "airtel", "8108": "airtel",
	"8109": "airtel",
	"8123": "airtel", "8124": "airtel", "8125": "airtel",
	"8126": "airtel", "8127": "airtel", "8128": "airtel",
	"8129": "airtel",
	"8147": "airtel", "8148": "airtel", "8149": "airtel",
	"8150": "airtel", "8151": "airtel", "8152": "airtel",
	"8153": "airtel", "8154": "airtel", "8155": "airtel",
	// 9xxx series
	"9400": "airtel", "9401": "airtel", "9402": "airtel",
	"9403": "airtel", "9404": "airtel", "9405": "airtel",
	"9406": "airtel", "9407": "airtel", "9409": "airtel", // 9408 → vi
	"9446": "airtel", "9447": "airtel", "9448": "airtel",
	"9449": "airtel",
	"9480": "airtel", "9481": "airtel", "9482": "airtel",
	"9483": "airtel", "9484": "airtel", "9485": "airtel",
	"9486": "airtel", "9487": "airtel", "9488": "airtel",
	"9489": "airtel",
	"9810": "airtel", "9811": "airtel", "9818": "airtel",
	"9871": "airtel", "9873": "airtel", // 9868 → bsnl
	"9899": "airtel", "9958": "airtel", "9953": "airtel",
	"9717": "airtel", "9718": "airtel", "9716": "airtel",
	"9891": "airtel", "9990": "airtel", "9999": "airtel",

	// ══ JIO ═════════════════════════════════════════
	// 6xxx series (Jio dominates 6xxx)
	"6000": "jio", "6001": "jio", "6002": "jio",
	"6003": "jio", "6004": "jio", "6005": "jio",
	"6006": "jio", "6007": "jio", "6008": "jio",
	"6009": "jio",
	"6100": "jio", "6101": "jio", "6102": "jio",
	"6103": "jio", "6104": "jio", "6105": "jio",
	"6106": "jio", "6107": "jio", "6108": "jio",
	"6109": "jio",
	// 7xxx series
	"7000": "jio", "7001": "jio", "7002": "jio",
	"7007": "jio", "7008": "jio", "7009": "jio",
	"7010": "jio", "7011": "jio", "7012": "jio",
	"7013": "jio", "7014": "jio", "7015": "jio",
	"7016": "jio", "7017": "jio", "7018": "jio",
	"7019": "jio",
	"7020": "jio", "7021": "jio", "7022": "jio",
	"7023": "jio", "7024": "jio", "7025": "jio",
	"7026": "jio", "7027": "jio", "7028": "jio",
	// 8xxx series
	"8104": "jio", "8169": "jio", "8369": "jio",
	"8668": "jio", "8669": "jio",
	"8975": "jio", "8976": "jio",
	"8779": "jio", "8828": "jio",
	// 9xxx series
	"9080": "jio", "9081": "jio", "9082": "jio",
	"9136": "jio", "9137": "jio",
	"9321": "jio", "9322": "jio", "9324": "jio", "9326": "jio",
	"9372": "jio", "9373": "jio", "9374": "jio",
	"9375": "jio", "9376": "jio", "9377": "jio",
	"9819": "jio", "9867": "jio", "9987": "jio",

	// ══ VI (Vodafone Idea) ══════════════════════════
	// 7xxx series
	"7700": "vi", "7701": "vi", "7702": "vi",
	"7703": "vi", "7704": "vi", "7705": "vi",
	"7706": "vi", "7707": "vi", "7708": "vi",
	"7709": "vi",
	// 8xxx series
	"8200": "vi", "8201": "vi", "8202": "vi",
	"8203": "vi", "8204": "vi", "8205": "vi",
	"8206": "vi", "8207": "vi", "8208": "vi",
	"8209": "vi",
	"8238": "vi",
	"8287": "vi", "8800": "vi",
	"8866": "vi", "8867": "vi",
	"8980": "vi",
	// 9xxx series
	"9033": "vi", "9099": "vi", "9106": "vi",
	"9173": "vi", "9227": "vi", "9228": "vi",
	"9229": "vi", "9265": "vi", "9316": "vi",
	"9317": "vi", "9319": "vi", "9328": "vi",
	"9408": "vi", "9427": "vi", "9429": "vi",
	"9712": "vi", "9714": "vi", "9726": "vi",
	"9727": "vi", "9825": "vi", "9898": "vi",
	"9909": "vi", "9913": "vi", "9924": "vi",
	"9925": "vi", "9971": "vi", "9979": "vi",

	// ══ BSNL ════════════════════════════════════════
	"9418": "bsnl", "9419": "bsnl", "9420": "bsnl",
	"9421": "bsnl", "9422": "bsnl", "9423": "bsnl",
	"9424": "bsnl", "9425": "bsnl", "9426": "bsnl",
	"9431": "bsnl", "9432": "bsnl", "9433": "bsnl",
	"9434": "bsnl", "9435": "bsnl", "9436": "bsnl",
	"9437": "bsnl", "9438": "bsnl", "9439": "bsnl",
	"9441": "bsnl", "9442": "bsnl", "9443": "bsnl",
	"9444": "bsnl", "9445": "bsnl",
	"9450": "bsnl", "9451": "bsnl", "9452": "bsnl",
	"9453": "bsnl", "9454": "bsnl", "9455": "bsnl",
	"9456": "bsnl", "9457": "bsnl", "9458": "bsnl",
	"9459": "bsnl",
	"9795": "bsnl", "9796": "bsnl", "9797": "bsnl",
	"9868": "bsnl", "9869": "bsnl",
}

var IndiaCarrierGateway = map[string]string{
	"airtel": "airtelmail.in",
	"jio":    "jio.com",
	"vi":     "vimail.in",
	"bsnl":   "bsnl.in",
}

// DetectIndiaCarrier returns carrier and email gateway for any Indian mobile number.
func DetectIndiaCarrier(e164 string) (carrier, gateway string) {
	digits := extractDigits(e164)

	local := digits
	if len(digits) > 10 && digits[:2] == "91" {
		local = digits[2:]
	}

	if len(local) < 4 {
		return "", ""
	}

	carrier = IndiaCarrierDB[local[:4]]
	if carrier == "" {
		carrier = IndiaCarrierDB[local[:3]]
	}
	if carrier == "" {
		return "", ""
	}

	gateway = IndiaCarrierGateway[carrier]
	return carrier, gateway
}
