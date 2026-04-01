package killkrill

// Hook is a placeholder for KillKrill integration
// The actual hook implementation would integrate with the logging adapter
type Hook struct {
	client *Client
}

// NewHook creates a new KillKrill hook
func NewHook(client *Client) *Hook {
	return &Hook{
		client: client,
	}
}

// Fire sends the log entry to KillKrill
// This is a placeholder implementation
func (h *Hook) Fire(entry interface{}) error {
	if h.client == nil || !h.client.config.Enabled {
		return nil
	}

	// Convert log entry to KillKrill format
	killKrillEntry := LogrusToKillKrill(entry)

	// Send to KillKrill (non-blocking)
	h.client.SendLog(killKrillEntry)

	return nil
}