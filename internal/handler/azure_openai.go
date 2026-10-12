package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// azureOpenAIAPIVersion is the Azure OpenAI API version used for chat completions.
const azureOpenAIAPIVersion = "2024-10-21"

// azureOpenAI is the Azure OpenAI resource the AI handlers call.
type azureOpenAI struct {
	endpoint string
	apiKey   string
}

// azureOpenAIFromEnv reads the resource from AZURE_OPENAI_ENDPOINT and AZURE_OPENAI_API_KEY.
func azureOpenAIFromEnv() (azureOpenAI, error) {
	az := azureOpenAI{
		endpoint: strings.TrimRight(strings.TrimSpace(os.Getenv("AZURE_OPENAI_ENDPOINT")), "/"),
		apiKey:   strings.TrimSpace(os.Getenv("AZURE_OPENAI_API_KEY")),
	}
	if az.endpoint == "" || az.apiKey == "" {
		return azureOpenAI{}, fmt.Errorf("Azure OpenAI not configured")
	}
	return az, nil
}

// azureDeployment returns the deployment named by envKey, or fallback when it is unset.
func azureDeployment(envKey, fallback string) string {
	if deployment := strings.TrimSpace(os.Getenv(envKey)); deployment != "" {
		return deployment
	}
	return fallback
}

// newChatRequest builds a chat completions request for deployment. The deployment
// picks the model, so payload carries no "model" field.
func (az azureOpenAI) newChatRequest(deployment string, payload map[string]interface{}) (*http.Request, error) {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		az.endpoint, url.PathEscape(deployment), azureOpenAIAPIVersion)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(payloadJSON))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("api-key", az.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}
