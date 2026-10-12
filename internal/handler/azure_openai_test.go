package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/studojo/jobs/internal/domain/resume"
)

// chatRequest is what the fake Azure OpenAI server received.
type chatRequest struct {
	method     string
	path       string
	apiVersion string
	header     http.Header
	body       map[string]interface{}
}

// fakeAzureOpenAI points the Azure env at a test server that answers with content
// as the first choice and passes on the request it received.
func fakeAzureOpenAI(t *testing.T, content string) <-chan chatRequest {
	t.Helper()
	seen := make(chan chatRequest, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := chatRequest{
			method:     r.Method,
			path:       r.URL.Path,
			apiVersion: r.URL.Query().Get("api-version"),
			header:     r.Header.Clone(),
		}
		if err := json.NewDecoder(r.Body).Decode(&got.body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		select {
		case seen <- got:
		default:
			t.Errorf("unexpected extra request to %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{"message": map[string]string{"role": "assistant", "content": content}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	// The trailing slash must not double up in the request path.
	t.Setenv("AZURE_OPENAI_ENDPOINT", srv.URL+"/")
	t.Setenv("AZURE_OPENAI_API_KEY", "test-key")
	t.Setenv("AZURE_OPENAI_MINI_DEPLOYMENT", "")
	t.Setenv("AZURE_OPENAI_CHAT_DEPLOYMENT", "")
	return seen
}

func TestAzureChatRequest(t *testing.T) {
	suggest := func(t *testing.T) {
		got, err := getAISuggestions(resume.Resume{Summary: "Builds things."}, "Engineer", "Go")
		if err != nil {
			t.Fatalf("getAISuggestions: %v", err)
		}
		if len(got) != 1 || got[0].Section != "summary" {
			t.Fatalf("suggestions = %+v, want the one the fake returned", got)
		}
	}
	optimize := func(t *testing.T) {
		got, err := optimizeResumeWithLLM(resume.Resume{Title: "Original"}, "Engineer", "Acme", "Go")
		if err != nil {
			t.Fatalf("optimizeResumeWithLLM: %v", err)
		}
		if got.Title != "Optimized" {
			t.Fatalf("title = %q, want the one the fake returned", got.Title)
		}
	}
	const suggestion = `[{"section":"summary","type":"rewrite","message":"m","severity":"info"}]`
	const optimized = `{"title":"Optimized"}`

	tests := []struct {
		name     string
		content  string
		call     func(t *testing.T)
		envKey   string // deployment override; empty keeps the default
		override string
		wantPath string
	}{
		{"suggest default", suggestion, suggest, "", "", "/openai/deployments/gpt-4o-mini/chat/completions"},
		{"suggest override", suggestion, suggest, "AZURE_OPENAI_MINI_DEPLOYMENT", "mini-test", "/openai/deployments/mini-test/chat/completions"},
		{"optimize default", optimized, optimize, "", "", "/openai/deployments/gpt-4o/chat/completions"},
		{"optimize override", optimized, optimize, "AZURE_OPENAI_CHAT_DEPLOYMENT", "chat-test", "/openai/deployments/chat-test/chat/completions"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seen := fakeAzureOpenAI(t, tt.content)
			if tt.envKey != "" {
				t.Setenv(tt.envKey, tt.override)
			}

			tt.call(t)

			var got chatRequest
			select {
			case got = <-seen:
			default:
				t.Fatal("no request reached the fake Azure OpenAI server")
			}
			if got.method != http.MethodPost || got.path != tt.wantPath {
				t.Errorf("request = %s %s, want POST %s", got.method, got.path, tt.wantPath)
			}
			if got.apiVersion != "2024-10-21" {
				t.Errorf("api-version = %q, want 2024-10-21", got.apiVersion)
			}
			if key := got.header.Get("api-key"); key != "test-key" {
				t.Errorf("api-key header = %q, want test-key", key)
			}
			if auth := got.header.Get("Authorization"); auth != "" {
				t.Errorf("Authorization header = %q, want none", auth)
			}
			if _, ok := got.body["model"]; ok {
				t.Errorf("body has a model field (%v), the deployment picks the model", got.body["model"])
			}
			if got.body["temperature"] != 0.3 {
				t.Errorf("temperature = %v, want 0.3", got.body["temperature"])
			}
			if msgs, _ := got.body["messages"].([]interface{}); len(msgs) != 2 {
				t.Errorf("messages = %v, want system and user", got.body["messages"])
			}
		})
	}
}

func TestAzureOpenAINotConfigured(t *testing.T) {
	t.Setenv("AZURE_OPENAI_ENDPOINT", "")
	t.Setenv("AZURE_OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "sk-retired") // the direct key is no longer read

	if _, err := getAISuggestions(resume.Resume{}, "", ""); err == nil || err.Error() != "Azure OpenAI not configured" {
		t.Errorf("getAISuggestions error = %v, want Azure OpenAI not configured", err)
	}

	in := resume.Resume{Title: "Original"}
	out, err := optimizeResumeWithLLM(in, "", "", "")
	if err == nil || err.Error() != "Azure OpenAI not configured" {
		t.Errorf("optimizeResumeWithLLM error = %v, want Azure OpenAI not configured", err)
	}
	if out.Title != in.Title {
		t.Errorf("optimizeResumeWithLLM returned title %q, want the input resume back", out.Title)
	}
}
