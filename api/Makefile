GO111MODULE = on
export GO111MODULE

api.pb.go: api.proto .FORCE
	go build google.golang.org/grpc/cmd/protoc-gen-go-grpc
	go build google.golang.org/protobuf/cmd/protoc-gen-go
	PATH="$(CURDIR):$(PATH)" protoc -I. -I.. --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative $<
	rm protoc-gen-go protoc-gen-go-grpc

.FORCE:
