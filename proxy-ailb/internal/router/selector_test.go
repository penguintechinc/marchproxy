package router

import "testing"

func TestParseModelSelector(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		provider string
		model    string
		ok       bool
	}{
		{"empty", "", "", "", false},
		{"anthropic opus", "Anthropic:Opus", "anthropic", "Opus", true},
		{"openai gpt4", "OpenAI:gpt-4o", "openai", "gpt-4o", true},
		{"ollama with colon", "Ollama:llama3.1:8b", "ollama", "llama3.1:8b", true},
		{"no colon", "JustAModel", "", "", false},
		{"empty provider", ":model", "", "", false},
		{"empty model", "Provider:", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model, ok := ParseModelSelector(tt.header)
			if ok != tt.ok {
				t.Errorf("ParseModelSelector(%q) ok = %v, want %v", tt.header, ok, tt.ok)
			}
			if ok {
				if provider != tt.provider {
					t.Errorf("provider = %q, want %q", provider, tt.provider)
				}
				if model != tt.model {
					t.Errorf("model = %q, want %q", model, tt.model)
				}
			}
		})
	}
}
