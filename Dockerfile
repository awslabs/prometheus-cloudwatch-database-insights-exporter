# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS build

WORKDIR /src

# Dependencies first, so a source-only change keeps the module cache warm.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY pkg ./pkg

# Static build: the final stage has no libc.
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/dbinsights-exporter ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/dbinsights-exporter /usr/local/bin/dbinsights-exporter

# Matches export.port in the default configuration.
EXPOSE 8081

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/dbinsights-exporter"]
