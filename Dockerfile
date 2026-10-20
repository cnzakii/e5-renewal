# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM oven/bun:1.4.3-alpine@sha256:629e17411f1f129dbec3af78d5af9c9f2a937435c80349437206c6b0b7422373 AS ui
WORKDIR /src/frontend
COPY frontend/package.json frontend/bun.lock frontend/bunfig.toml frontend/.bun-version ./
RUN --mount=type=cache,target=/root/.bun/install/cache \
    test "$(bun --version)" = "$(cat .bun-version)" && bun install --frozen-lockfile
COPY frontend/ ./
RUN bun run build

FROM --platform=$BUILDPLATFORM golang:1.27.2-alpine@sha256:f92b6ef800e499660581efdabdf25d9d817a9d124eaf900924f0504e7e27e12d AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    test "$(GOTOOLCHAIN=local go env GOVERSION)" = "$(awk '/^toolchain / {print $2}' go.mod)" && go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/ web/
COPY --from=ui /src/frontend/dist/ web/dist/
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X main.version=$VERSION" -o /out/e5-renewal ./cmd/e5-renewal
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS runtime
WORKDIR /app
COPY --from=build /out/e5-renewal /app/e5-renewal
COPY --from=build --chown=65532:65532 /out/data /data
COPY LICENSE /licenses/
ENV E5_LISTEN=0.0.0.0:8080 E5_DATA_DIR=/data
EXPOSE 8080
ENTRYPOINT ["/app/e5-renewal"]
