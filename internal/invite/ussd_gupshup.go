package invite

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func SendUSSDGupshup(toE164 string, callerNumber string, inviteURL string) error {
	apiKey := os.Getenv("GUPSHUP_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GUPSHUP_API_KEY not set")
	}

	message := fmt.Sprintf(
		"OpenCall: %s wants to call you free. Visit %s to answer or reply ACCEPT",
		callerNumber, inviteURL,
	)

	formData := url.Values{}
	formData.Set("method", "sendMessage")
	formData.Set("send_to", toE164)
	formData.Set("msg", message)
	formData.Set("msg_type", "TEXT")
	formData.Set("userid", os.Getenv("GUPSHUP_USERID"))
	formData.Set("auth_scheme", "plain")
	formData.Set("password", apiKey)
	formData.Set("v", "1.1")
	formData.Set("format", "text")

	req, err := http.NewRequest("POST",
		"https://enterprise.smsgupshup.com/GatewayAPI/rest",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("Gupshup error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
