-include Makefile.options
#####################################################################################
## print usage information
help:
	@echo 'Usage:'
	@cat ${MAKEFILE_LIST} | grep -e "^## " -A 1 | grep -v '\-\-' | sed 's/^##//' | cut -f1 -d":" | \
		awk '{info=$$0; getline; print "  " $$0 ": " info;}' | column -t -s ':' | sort 
.PHONY: help
#####################################################################################
generate:
	go get github.com/petergtz/pegomock/...
	go generate ./...

renew-async-api:
	go get github.com/airenas/async-api@$$(cd ../async-api;git rev-parse HEAD)	
#####################################################################################
## call units tests
test/unit: 
	go test -v -race -count=1 ./...
.PHONY: test/unit
#####################################################################################
## code vet and lint
test/lint: 
	go vet ./...
	go get -u golang.org/x/lint/golint
	golint -set_exit_status ./...
.PHONY: test/lint
#####################################################################################


