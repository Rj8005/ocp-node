package invite

import "strings"

type USSDGateway int

const (
	GatewayAfricasTalking USSDGateway = iota
	GatewayGupshup
	GatewayNone
)

type GatewayConfig struct {
	Gateway     USSDGateway
	CountryName string
	CountryCode string
}

func DetectUSSDGateway(e164 string) GatewayConfig {
	africasTalkingPrefixes := map[string]string{
		"+254": "Kenya",
		"+234": "Nigeria",
		"+233": "Ghana",
		"+256": "Uganda",
		"+255": "Tanzania",
		"+250": "Rwanda",
		"+251": "Ethiopia",
		"+237": "Cameroon",
		"+225": "Cote dIvoire",
		"+260": "Zambia",
		"+265": "Malawi",
		"+258": "Mozambique",
		"+267": "Botswana",
		"+263": "Zimbabwe",
		"+264": "Namibia",
		"+268": "Eswatini",
		"+266": "Lesotho",
		"+221": "Senegal",
		"+223": "Mali",
		"+226": "Burkina Faso",
		"+227": "Niger",
		"+229": "Benin",
		"+228": "Togo",
		"+224": "Guinea",
		"+245": "Guinea-Bissau",
		"+232": "Sierra Leone",
		"+231": "Liberia",
		"+220": "Gambia",
		"+222": "Mauritania",
	}

	gupshupPrefixes := map[string]string{
		"+91":  "India",
		"+880": "Bangladesh",
		"+92":  "Pakistan",
		"+94":  "Sri Lanka",
		"+977": "Nepal",
	}

	for prefix, country := range africasTalkingPrefixes {
		if strings.HasPrefix(e164, prefix) {
			return GatewayConfig{GatewayAfricasTalking, country, prefix}
		}
	}

	for prefix, country := range gupshupPrefixes {
		if strings.HasPrefix(e164, prefix) {
			return GatewayConfig{GatewayGupshup, country, prefix}
		}
	}

	return GatewayConfig{GatewayNone, "Unknown", ""}
}
