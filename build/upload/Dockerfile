#########################################################################################
## docker will invoke this file from ../.. dir in order to access code
#########################################################################################
FROM golang:1.21-alpine AS builder

ARG BUILD_VERSION=0.1

WORKDIR /go/src/
ENV CGO_ENABLED=0

# precache go.mod dependencies
COPY go.mod /go/src/
COPY go.sum /go/src/
COPY cmd /go/src/cmd
COPY internal /go/src/internal

RUN CGO_ENABLED=0 go build -o /go/bin/upload -ldflags "-X main.version=$BUILD_VERSION" cmd/upload/main.go
#####################################################################################
FROM alpine:3.15 AS runner

EXPOSE 8000


ENV UID=999
RUN addgroup app && adduser -S -G app app --uid $UID app

WORKDIR /app
RUN mkdir -p /app && chown -R app:app /app
RUN mkdir -p /data && chown -R app:app /data   

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
