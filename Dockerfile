FROM golang:1.25-alpine3.22 AS builder

WORKDIR /app

RUN apk add --no-cache --virtual .build-deps \
    git \
    gcc \
    musl-dev

COPY go.mod go.sum ./
COPY tmp/main.go tmp/go.mod ./tmp/
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o julia

RUN apk del .build-deps

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache \
    ffmpeg \
    bash \
    file \
    coreutils \
    gawk \
    neofetch \
    mediainfo \
    wget \
    tar \
    yt-dlp

ENV GOLANG_VERSION=1.25.0

RUN wget https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz -O /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
 
RUN apk add --no-cache "$PKG_A"

COPY --from=builder /app/julia /app/cover_gen.sh ./
COPY --from=builder /app/assets /app/assets

RUN chmod +x /app/julia /app/cover_gen.sh

ENV GOCACHE=/app/.cache/go-build
ENV GOMODCACHE=/app/.cache/go-mod

RUN mkdir -p /app/.cache/go-build /app/.cache/go-mod /app/tmp

COPY --from=builder /app/tmp/go.mod /app/tmp/
COPY --from=builder /app/tmp/main.go /app/tmp/
RUN cd /app/tmp && go mod tidy && go get -u github.com/amarnathcjd/gogram@dev

ENTRYPOINT ["/app/julia"]
