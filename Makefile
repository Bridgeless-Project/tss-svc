protogen:
	cd proto && \
	buf generate deposit --template=./templates/deposit.yaml --config=buf.yaml && \
	buf generate p2p --template=./templates/p2p.yaml --config=buf.yaml && \
	buf generate api --template=./templates/api.yaml --config=buf.yaml

account:
	go run main.go helpers generate cosmos-account

preparams-f:
	go run main.go helpers generate preparams -o file --path=./internal/config/preparams.json
