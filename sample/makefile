PROTO_SRC_FILES=$(shell find ./proto -type f -name "*.proto" | sed 's/\/proto//g')

.PHONY: proto
proto:
	cd proto; \
	protoc -I=. -I=${GOPATH}/src/github.com/protobuf \
		--gofast_out=paths=source_relative:../pb  \
		$(PROTO_SRC_FILES)

lint:
	go vet ./...
