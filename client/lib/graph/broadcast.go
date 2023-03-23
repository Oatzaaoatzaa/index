package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func Broadcast(graphUrl, txRaw string) error {
	jsonData := map[string]interface{}{
		"query": broadcastQuery,
		"variables": map[string]string{
			"raw": txRaw,
		},
	}
	jsonValue, err := json.Marshal(jsonData)
	if err != nil {
		return fmt.Errorf("error marshaling json for broadcast; %w", err)
	}
	request, err := http.NewRequest("POST", graphUrl, bytes.NewBuffer(jsonValue))
	if err != nil {
		return fmt.Errorf("error making a new request for broadcast failed; %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: time.Second * 10}
	resp, err := client.Do(request);
	if err != nil {
		return fmt.Errorf("error the HTTP request for broadcast failed with error; %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("error the HTTP request for broadcast failed with status code: %s", resp.Status)
	}
	return nil
}

const broadcastQuery = `mutation ($raw: String!) {
	broadcast(raw: $raw)
}`
