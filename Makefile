generate:
	protoc --proto_path=proto proto/*.proto --go_out=. --go_grpc_out=.