package grpc

import (
	"github.com/penguintech/marchproxy/proxy-rtmp/internal/logging"
)

var logger *logging.LogrusAdapter

func init() {
	var err error
	logger, err = logging.NewLogrusAdapter("grpc")
	if err != nil {
		panic("failed to initialize grpc logger: " + err.Error())
	}
}
