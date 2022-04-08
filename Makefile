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
## run integration tests
test/integration: 
	cd testing/integration && ( $(MAKE) -j1 test/integration clean || ( $(MAKE) clean; exit 1; ))
.PHONY: test/integration
#####################################################################################
## code vet and lint
test/lint: 
	go vet ./...
	go install golang.org/x/lint/golint@latest
	golint -set_exit_status ./...
.PHONY: test/lint
#####################################################################################
## build docker for provided service
docker/%/build: 
	cd build/$* && $(MAKE) dbuild
.PHONY: docker/*/build
#####################################################################################
## push docker for provided service
docker/%/push: 
	cd build/$* && $(MAKE) dpush
.PHONY: docker/*/push
#####################################################################################
## scan docker for provided service
docker/%/scan: 
	cd build/$* && $(MAKE) dscan
.PHONY: docker/*/scan
#####################################################################################
## cleans temporary data
clean: clean/clean clean/result
	go mod tidy -compat=1.17
	go clean
.PHONY: clean
clean/%:
	cd build/$* && $(MAKE) clean
#####################################################################################


