# 防止命令行参数被误认为是目标
%:
	@:

.PHONY: gen
gen: api/v1/api.proto
	go build github.com/gogo/protobuf/protoc-gen-gogofaster
	-PATH="$(CURDIR):$(PATH)" protoc  --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative,require_unimplemented_servers=false $<
	-PATH="$(CURDIR):$(PATH)" protoc --gogofaster_out=paths=source_relative:. $<
	-go fmt ./...
	rm protoc-gen-gogofaster
	-protoc-go-inject-tag -input=api/v1/*.pb.go
