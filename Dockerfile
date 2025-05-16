FROM golang:1.24.3-alpine AS build

WORKDIR /build

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build  \
    -o /bin/auto-mcp ./cmd/auto-mcp

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=build /bin/auto-mcp .

COPY config.yaml .


ENTRYPOINT ["./auto-mcp"]
CMD ["--mode=stdio"]
