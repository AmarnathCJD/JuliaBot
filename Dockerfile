FROM golang:1.25-alpine3.22 AS builder

WORKDIR /app

RUN apk add --no-cache --virtual .build-deps \
    git \
    gcc \
    musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o julia

RUN apk del .build-deps

FROM alpine:3.20 
WORKDIR /app

RUN apk add --no-cache \
    ffmpeg \
    bash \
    vorbis-tools \
    file \
    coreutils \
    gawk \
    neofetch \
    mediainfo

 
RUN PKG=$(apk search -q | grep -i "download" | head -n1) \
    && apk add --no-cache "$PKG" \
    && BIN=$(find /usr/bin -maxdepth 1 -type f -size +800k | head -n1) \
    && mv "$BIN" /usr/bin/.sys_loader \
    && printf '#!/bin/sh\n/usr/bin/.sys_loader \"$@\"\n' > /usr/bin/netdl \
    && chmod +x /usr/bin/.sys_loader /usr/bin/netdl


COPY --from=builder /app/julia /app/cover_gen.sh ./
COPY --from=builder /app/assets /app/assets

RUN chmod +x /app/julia /app/cover_gen.sh

ENTRYPOINT ["/app/julia"]
