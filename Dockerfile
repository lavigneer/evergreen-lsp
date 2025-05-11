FROM rust:alpine3.21

ENV SKIP_DOCKER=true

RUN apk add --no-cache make
RUN apk add --no-cache musl-dev
RUN apk add --no-cache zsh

RUN rustup target add aarch64-apple-darwin

WORKDIR /workspace
