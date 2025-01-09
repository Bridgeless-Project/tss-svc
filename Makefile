protogen-p2p:
	cd proto/p2p && buf generate

account:
	go run main.go helpers generate cosmos-account

preparams-f:
	go run main.go helpers generate preparams -o file --path=./internal/config/preparams.json
