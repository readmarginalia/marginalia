FROM golang:1.25-alpine AS build
WORKDIR /src

ARG VERSION=dev

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build \
    -ldflags "-X 'marginalia/internal/buildinfo.Version=${VERSION}'" \
    -o /marginalia ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /marginalia /marginalia
USER 1001
ENTRYPOINT ["/marginalia"]
