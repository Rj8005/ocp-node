package invite

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type ATUSSDRequest struct {
	Username  string
	To        string
	Message   string
	SessionID string
}

type ATUSSDResponse struct {
	SMSMessageData struct {
		Message    string `json:"Message"`
		Recipients []struct {
			StatusCode int    `json:"statusCode"`
			Number     string `json:"number"`
			Status     string `json:"status"`
		} `json:"Recipients"`
	} `json:"SMSMessageData"`
}

func SendUSSDAfricasTalking(toE164 string, callerNumber string, inviteURL string, sessionID string) error {
	apiKey := os.Getenv("AT_API_KEY")
	username := os.Getenv("AT_USERNAME")
	if apiKey == "" || username == "" {
		return fmt.Errorf("AT_API_KEY or AT_USERNAME not set")
	}

	message := fmt.Sprintf(
		"CON OpenCall\nFree call from %s\n\n1. Accept call now\n2. Decline\n3. Get link: %s",
		callerNumber, inviteURL,
	)

	formData := url.Values{}
	formData.Set("username", username)
	formData.Set("phoneNumber", toE164)
	formData.Set("sessionId", sessionID)
	formData.Set("text", message)

	req, err := http.NewRequest("POST",
		"https://api.africastalking.com/version1/ussd/send",
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("apiKey", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result ATUSSDResponse
	json.Unmarshal(body, &result)

	if resp.StatusCode != 200 {
		return fmt.Errorf("AT API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
