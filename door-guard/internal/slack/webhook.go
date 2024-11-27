package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"k-wa-wa/door-guard/internal/env"
	"net/http"
)

type reqBody struct {
	Text string `json:"text"`
}

func PostMessage(text string) error {
	reqBodyBytes, err := json.Marshal(reqBody{
		Text: text,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		env.SLACK_WEBHOOK_URL,
		bytes.NewBuffer(reqBodyBytes),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message, status: %s", res.Status)
	}

	return nil
}
