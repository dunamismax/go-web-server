# syntax=docker/dockerfile:1

# Build stage. The generated code (sqlc, templ, CSS) is checked in, so the
# image needs nothing but the Go toolchain.
FROM golang:1.27 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# The build context has no .git directory, so the version is passed in. See
# the "docker" target in the Makefile.
ARG VERSION=""
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
      -ldflags="-s -w -X github.com/dunamismax/go-web-server/internal/version.override=${VERSION}" \
      -o /out/server ./cmd/server

# Runtime stage: a static binary on distroless, running as a non-root user.
# The image has no shell or package manager.
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/server /server

ENV APP_ENV=production \
    HTTP_ADDR=:8080
EXPOSE 8080
USER nonroot:nonroot

# The binary probes itself; distroless has no curl or wget.
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["/server", "healthcheck"]

ENTRYPOINT ["/server"]
