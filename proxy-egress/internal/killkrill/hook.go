package killkrill

import (
	"github.com/sirupsen/logrus"
)

// Hook implements the logrus.Hook interface to send logs to KillKrill
type Hook struct {
	client *Client
}

// NewHook creates a new KillKrill logrus hook
func NewHook(client *Client) *Hook {
	return &Hook{
		client: client,
	}
}

// Levels returns the log levels that this hook handles
func (h *Hook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire sends the log entry to KillKrill
func (h *Hook) Fire(entry *logrus.Entry) error {
	if h.client == nil || !h.client.config.Enabled {
		return nil
	}

	// Convert logrus entry to KillKrill format
	killKrillEntry := LogrusToKillKrill(entry)

	// Send to KillKrill (non-blocking)
	h.client.SendLog(killKrillEntry)

	return nil
}