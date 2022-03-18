build:
	go build

tidy:
	go mod tidy

protoc:
	protoc --go_out=./protocol  protocol/comms.proto
	mv protocol/github.com/decentraland/nats-test/protocol/comms.pb.go protocol/
	rm -r protocol/github.com/

fmt:
	go fmt main.go
