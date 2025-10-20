SOME_PROTO_FILES = $(shell find somestub -name "*.proto")

.PHONY: install
install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	@echo "All protoc plugins installed!"

.PHONY: generate
generate:
	protoc --proto_path=./somestub \
		   --proto_path=./proto3ps \
		   --go_out=paths=source_relative:./somestub \
		   --go-http_out=paths=source_relative:./somestub \
		   --go-grpc_out=paths=source_relative:./somestub \
		   $(SOME_PROTO_FILES)
	@echo "Proto code generation complete!"

.PHONY: clean
clean:
	rm -f somestub/*.pb.go
	@echo "Cleanup complete!"

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  install  - Install protoc plugins (go, grpc, http)"
	@echo "  generate - Generate Go code from proto files"
	@echo "  clean    - Remove generated files"
	@echo "  help     - Show this help message"
