package invite

import (
	"net/url"
	"strings"
)

// Channel describes a messaging platform that can carry an OCP invite link.
type Channel struct {
	Name             string
	DeepLinkTemplate string   // placeholders: {number} (digits only), {message} (URL-encoded)
	APIAvailable     bool     // true when the channel exposes a server-side send API
	GlobalFallback   bool     // true when usable worldwide without country restriction
	Countries        []string // ISO 3166-1 alpha-2 codes where this channel is dominant
}

// ── Channel definitions ───────────────────────────────────────────────────────

var WhatsApp = Channel{
	Name:             "WhatsApp",
	DeepLinkTemplate: "https://wa.me/{number}?text={message}",
	GlobalFallback:   true,
}

var Telegram = Channel{
	Name:             "Telegram",
	DeepLinkTemplate: "https://t.me/+{number}",
	Countries:        []string{"IR", "RU", "UA", "BY", "UZ"},
}

var Viber = Channel{
	Name:             "Viber",
	DeepLinkTemplate: "viber://chat?number=%2B{number}",
	Countries:        []string{"RU", "UA", "BY", "RS", "GR", "IL", "PH"},
}

var LINE = Channel{
	Name:             "LINE",
	DeepLinkTemplate: "https://line.me/ti/p/~{number}",
	Countries:        []string{"JP", "TH", "TW", "ID", "VN"},
}

var KakaoTalk = Channel{
	Name:             "KakaoTalk",
	DeepLinkTemplate: "kakaoplus://friend/{number}",
	Countries:        []string{"KR"},
}

var SMSUniversal = Channel{
	Name:             "SMS",
	DeepLinkTemplate: "sms:{number}?body={message}",
	GlobalFallback:   true,
}

var TextBelt = Channel{
	Name:           "TextBelt",
	APIAvailable:   true,
	GlobalFallback: true,
}

var Signal = Channel{
	Name:             "Signal",
	DeepLinkTemplate: "https://signal.me/#p/+{number}",
	Countries:        []string{"US", "DE", "NL", "CH", "AT"},
}

var Zalo = Channel{
	Name:             "Zalo",
	DeepLinkTemplate: "https://zalo.me/{number}",
	Countries:        []string{"VN"},
}

// WeChat has no universal deep-link scheme; BuildDeepLink returns "" for it.
var WeChat = Channel{
	Name:             "WeChat",
	DeepLinkTemplate: "",
	Countries:        []string{"CN"},
}

// ── Priority map ──────────────────────────────────────────────────────────────

// CountryChannels maps ISO 3166-1 alpha-2 country codes to an ordered slice of
// preferred invite channels, from most-likely-installed to least.
// "DEFAULT" is used for any country not explicitly listed.
var CountryChannels = map[string][]Channel{
	"CN":      {WeChat, SMSUniversal, TextBelt},
	"JP":      {LINE, WhatsApp, SMSUniversal, Telegram},
	"KR":      {KakaoTalk, WhatsApp, SMSUniversal, Telegram},
	"RU":      {Telegram, Viber, WhatsApp, SMSUniversal},
	"IR":      {Telegram, WhatsApp, SMSUniversal},
	"UA":      {Viber, Telegram, WhatsApp, SMSUniversal},
	"VN":      {Zalo, Viber, WhatsApp, SMSUniversal},
	"PH":      {Viber, WhatsApp, SMSUniversal},
	"US":      {SMSUniversal, WhatsApp, Signal, Telegram},
	"GB":      {WhatsApp, SMSUniversal, Telegram, Signal},
	"IN":      {WhatsApp, Telegram, SMSUniversal},
	"DE":      {WhatsApp, Signal, Telegram, SMSUniversal},
	"BR":      {WhatsApp, Telegram, SMSUniversal},
	"ID":      {WhatsApp, LINE, Telegram, SMSUniversal},
	"DEFAULT": {WhatsApp, Telegram, Viber, SMSUniversal, TextBelt},
}

// ── Functions ─────────────────────────────────────────────────────────────────

// GetChannels returns the ordered channel list for countryCode.
// Falls back to the DEFAULT list for unknown country codes.
func GetChannels(countryCode string) []Channel {
	if channels, ok := CountryChannels[countryCode]; ok {
		return channels
	}
	return CountryChannels["DEFAULT"]
}

// BuildDeepLink expands ch.DeepLinkTemplate for the given phone number and
// invite URL. Returns an empty string when the channel has no deep-link scheme
// (e.g. WeChat).
//
//   - {number}  → e164 with the leading '+' stripped
//   - {message} → URL-encoded "Call me free on OpenCall: <inviteURL>"
func BuildDeepLink(ch Channel, e164 string, inviteURL string) string {
	if ch.DeepLinkTemplate == "" {
		return ""
	}
	number := strings.TrimPrefix(e164, "+")
	message := url.QueryEscape("Call me free on OpenCall: " + inviteURL)

	result := ch.DeepLinkTemplate
	result = strings.ReplaceAll(result, "{number}", number)
	result = strings.ReplaceAll(result, "{message}", message)
	return result
}

// IsAPIChannel reports whether ch supports server-side delivery via an API,
// as opposed to client-side deep links only.
func IsAPIChannel(ch Channel) bool {
	return ch.APIAvailable
}
