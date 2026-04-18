package webrtc

import (
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/logging"
)

var logger *logging.LogrusAdapter

func init() {
	var err error
	logger, err = logging.NewLogrusAdapter("webrtc")
	if err != nil {
		panic("failed to initialize webrtc logger: " + err.Error())
	}
}
