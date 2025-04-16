FROM rust:alpine3.21

ENV SKIP_DOCKER=true

RUN apk add --no-cache make

WORKDIR /workspace
