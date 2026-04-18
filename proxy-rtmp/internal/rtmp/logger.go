package rtmp

import (
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/logging"
)

var logger *logging.LogrusAdapter

func init() {
	var err error
	logger, err = logging.NewLogrusAdapter("rtmp")
	if err != nil {
		panic("failed to initialize rtmp logger: " + err.Error())
	}
}
