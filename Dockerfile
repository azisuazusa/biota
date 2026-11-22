# syntax=docker/dockerfile:1

# ---- build: templ generate + tailwind build + go build ----
FROM golang:1.26-alpine AS build
ARG TEMPL_VERSION=v0.3.977
ARG TAILWIND_VERSION=v4.3.5
ARG TARGETARCH
RUN apk add --no-cache curl ca-certificates
RUN go install github.com/a-h/templ/cmd/templ@${TEMPL_VERSION}
RUN case "${TARGETARCH:-amd64}" in \
      arm64) TW_ARCH=arm64 ;; \
      *) TW_ARCH=x64 ;; \
    esac && \
    curl -sSfL -o /usr/local/bin/tailwindcss \
      "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${TW_ARCH}-musl" && \
    chmod +x /usr/local/bin/tailwindcss

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN templ generate && \
    tailwindcss -i static/css/input.css -o static/css/app.css --minify && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./app/http && \
    mkdir -p /out/data

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /server
COPY --from=build /src/static /static
# Pre-create /data owned by nonroot so a fresh named volume is writable.
COPY --from=build --chown=nonroot:nonroot /out/data /data
ENV DATA_DIR=/data PORT=8080 MAX_UPLOAD_MB=10
VOLUME /data
EXPOSE 8080
USER nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 CMD ["/server", "-healthcheck"]
ENTRYPOINT ["/server"]
