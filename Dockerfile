FROM golang:1.25-alpine AS build
WORKDIR /src

ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags "-X 'marginalia/internal/buildinfo.Version=${VERSION}'" \
    -o /marginalia ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /marginalia /marginalia
ENTRYPOINT ["/marginalia"]
