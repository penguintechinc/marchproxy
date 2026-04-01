module github.com/PenguinTech/MarchProxy/proxy-alb

go 1.24

require (
	github.com/penguintechinc/penguin-libs/packages/go-common v0.0.0-20260311183616-aa9e846acf39
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.60.1
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240108191215-35c7eff3a6b1 // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/PenguinTech/MarchProxy/proto => ../proto
