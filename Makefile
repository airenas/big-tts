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
	go install github.com/petergtz/pegomock/...@latest
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
	go install golang.org/x/lint/golint@latest
	golint -set_exit_status ./...
.PHONY: test/lint
#####################################################################################
## build docker for provided service
build/%/dbuild: 
	cd build/$* && $(MAKE) dbuild
.PHONY: build/*/dbuild
#####################################################################################
## push docker for provided service
build/%/dpush: 
	cd build/$* && $(MAKE) dpush
.PHONY: build/*/dpush
#####################################################################################
## scan docker for provided service
build/%/dscan: 
	cd build/$* && $(MAKE) dscan
.PHONY: build/*/dscan
#####################################################################################
## cleans temporary data
clean: clean/synthesize clean/clean clean/inform clean/result clean/status clean/upload
	go mod tidy
	go clean
.PHONY: clean
clean/%:
	cd build/$* && $(MAKE) clean
#####################################################################################


