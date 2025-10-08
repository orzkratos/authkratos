SOME_PROTO_FILES = $(shell find somestub -name "*.proto")

.PHONY: proto
proto:
	protoc --proto_path=./somestub \
		   --proto_path=./proto3ps \
		   --go_out=paths=source_relative:./somestub \
		   --go-http_out=paths=source_relative:./somestub \
		   --go-grpc_out=paths=source_relative:./somestub \
		   $(SOME_PROTO_FILES)
