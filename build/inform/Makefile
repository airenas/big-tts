-include ../../version 
#####################################################################################
service=airenas/big-tts-inform
version?=$(inform_version)
file=build/inform/Dockerfile
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
