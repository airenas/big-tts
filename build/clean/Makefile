-include ../../version 
#####################################################################################
service=airenas/big-tts-clean
version?=$(clean_version)
file=build/clean/Dockerfile
########### DOCKER ##################################################################
tag=$(service):$(version)

dbuild:
	cd ../../ && docker build -t $(tag) --build-arg BUILD_VERSION=$(version) -f $(file) .

dpush: dbuild
	docker push $(tag)

dscan: dbuild
	docker scan --accept-license $(tag)	
#####################################################################################
.PHONY: dbuild dpush dscan
