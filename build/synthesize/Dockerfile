FROM iron/base:3.14.0 as runner

RUN apk add ffmpeg

## test ffmpeg
RUN ffmpeg -version

FROM runner

ARG BUILD_VERSION=0.1

LABEL org.opencontainers.image.version=$BUILD_VERSION \
      org.opencontainers.image.authors="airenass@gmail.com" \
      name="big-tts-upload" \
      maintainer="airenass@gmail.com" \
      vendor="airenass@gmail.com" \
      version=$BUILD_VERSION \
      release=$BUILD_VERSION \
      summary="This image is used to host the Big TTS Synthesize worker service" \
      description="This image is used to host the Big TTS Synthesize worker service" 

WORKDIR /app
VOLUME /data

ENV UID=999

RUN addgroup app && adduser -S -G app --uid $UID app
RUN mkdir -p /app && chown -R app:app /app   
RUN mkdir -p /data && chown -R app:app /data 

COPY data/synthesize /app/
COPY config.yaml /app/

RUN chown app:app /app/* /app
 
USER app

ENTRYPOINT ["./synthesize"]
