package transcode

import (
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/logging"
)

var logger *logging.LogrusAdapter

func init() {
	var err error
	logger, err = logging.NewLogrusAdapter("transcode")
	if err != nil {
		panic("failed to initialize transcode logger: " + err.Error())
	}
}
