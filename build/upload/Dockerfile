#########################################################################################
## docker will invoke this file from ../.. dir in order to access code
#########################################################################################
FROM golang:1.17-alpine as builder

ARG BUILD_VERSION=0.1

WORKDIR /go/src/
ENV CGO_ENABLED=0

# precache go.mod dependencies
COPY go.mod /go/src/
COPY go.sum /go/src/
RUN go mod download

# copy src
COPY . /go/src

RUN CGO_ENABLED=0 go build -o /go/bin/upload -ldflags "-X main.version=$BUILD_VERSION" cmd/upload/main.go
#####################################################################################
FROM iron/base:3.14.0 as runner

EXPOSE 8000


ENV UID=999
RUN addgroup app && adduser -S -G app app --uid $UID app

WORKDIR /app
RUN mkdir -p /app && chown -R app:app /app   

ARG BUILD_VERSION=0.1

LABEL org.opencontainers.image.version=$BUILD_VERSION \
      org.opencontainers.image.authors="airenass@gmail.com" \
      name="big-tts-upload" \
      maintainer="airenass@gmail.com" \
      vendor="airenass@gmail.com" \
      version=$BUILD_VERSION \
      release=$BUILD_VERSION \
      summary="This image is used to host the Big TTS Upload service" \
      description="This image is used to host the Big TTS Upload service" 

COPY --from=builder /go/bin/upload /app/
COPY build/upload/config.yaml /app/

RUN chown app:app /app/* /app
 
USER app

ENTRYPOINT ["./upload"]
