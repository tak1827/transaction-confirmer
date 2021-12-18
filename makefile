PROTO_SRC_FILES=$(shell find ./proto -type f -name "*.proto" | sed 's/\/proto//g')

.PHONY: proto
proto:
	cd proto; \
	protoc -I=. -I=${GOPATH}/src/github.com/protobuf \
		--gofast_out=paths=source_relative:../pb  \
		$(PROTO_SRC_FILES)

lint:
	go vet ./...

fmt:
	gofmt -w -l .

test:
	go test ./... -race

chain:
	ganache-cli -p 8545 -i 1010 -h localhost -l 30000000 \
		--account '0xd1c71e71b06e248c8dbe94d49ef6d6b0d64f5d71b1e33a0f39e14dadb070304a,1000000000000000000000' \
		--account '0x8179ce3d00ac1d1d1d38e4f038de00ccd0e0375517164ac5448e3acc847acb34,1000000000000000000000' \
		--account '0xdf38daebd09f56398cc8fd699b72f5ea6e416878312e1692476950f427928e7d,1000000000000000000000' \
		--account '0x97d12403ffc2faa3660730ae58bca14a894ebd78b4d8207d22083554ae96be5c,1000000000000000000000' \
		--account '0x71c64befc3dfd761a94cdcd1ce3e7603ea19cccdde4ac3428818821863e60481,1000000000000000000000' \
		--account '0x70d46b61473e44be3e4a438c8aa373a795eb8ee0155993776b51062c59353918,1000000000000000000000' \
		--account '0xe503cceff655bfb9075de9ff5bb3aa84aec08cf426c5fa87c1bec65ab5b975bc,1000000000000000000000' \
		--account '0x7663e0a3bb5b39b233726ea6d4939fb9477a8c24b4b289564f24fb8b651d4c25,1000000000000000000000' \
		--account '0x27f5756660416f3f1469b296ee4ee579b9a19167a15337b44e34e1f8221f4bd7,1000000000000000000000' \
		--account '0x432494ee8a9af04064bd4bee7419c7e7023ec9fcdb9d1b7a6d02289f62545dd7,1000000000000000000000'

