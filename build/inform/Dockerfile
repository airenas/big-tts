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
COPY cmd /go/src/cmd
COPY internal /go/src/internal

RUN CGO_ENABLED=0 go build -o /go/bin/inform -ldflags "-X main.version=$BUILD_VERSION" cmd/inform/main.go
#####################################################################################
FROM alpine:3.15 as runner

ENV UID=999
RUN addgroup app && adduser -S -G app app --uid $UID app

WORKDIR /app
RUN mkdir -p /app && chown -R app:app /app   

RUN apk --no-cache add tzdata

ARG BUILD_VERSION=0.1

LABEL org.opencontainers.image.version=$BUILD_VERSION \
      org.opencontainers.image.authors="airenass@gmail.com" \
      name="big-tts-inform" \
      maintainer="airenass@gmail.com" \
      vendor="airenass@gmail.com" \
      version=$BUILD_VERSION \
      release=$BUILD_VERSION \
      summary="This image is used to host the Big TTS Inform worker service" \
      description="This image is used to host the Big TTS Inform worker service" 

COPY --from=builder /go/bin/inform /app/
COPY build/inform/config.yaml /app/
COPY build/inform/mail.tmpl /app/template/

RUN chown app:app /app/* /app
 
USER app

ENTRYPOINT ["./inform"]
