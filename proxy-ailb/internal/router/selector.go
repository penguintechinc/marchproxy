package router

import "strings"

// ParseModelSelector parses the X-Model-Selector header value.
//
// Format: "Provider:ModelName"
// Examples:
//
//	"Anthropic:claude-3-opus-20240229"  -> ("anthropic", "claude-3-opus-20240229", true)
//	"Ollama:llama3.1:8b"               -> ("ollama", "llama3.1:8b", true)
//	"gpt-4"                            -> ("", "gpt-4", false)
//	""                                 -> ("", "", false)
//
// The provider portion is always lowercased. If no colon is present or the
// header is empty, ok is false and model is the raw header value.
func ParseModelSelector(header string) (provider, model string, ok bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", "", false
	}

	// Find the first colon that separates provider from model.
	// Model names themselves may contain colons (e.g. "llama3.1:8b"),
	// so we only split on the first colon.
	idx := strings.Index(header, ":")
	if idx <= 0 {
		// No colon, or colon at position 0 — treat as bare model name.
		return "", header, false
	}

	provider = strings.ToLower(strings.TrimSpace(header[:idx]))
	model = strings.TrimSpace(header[idx+1:])

	if model == "" {
		return "", header, false
	}

	return provider, model, true
}
