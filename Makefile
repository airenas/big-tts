generate:
	go get github.com/petergtz/pegomock/...
	go generate ./...

renew-async-api:
	go get github.com/airenas/async-api@$$(cd ../async-api;git rev-parse HEAD)	


