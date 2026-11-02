# ---- build ----
FROM golang:1.27-alpine AS builder
WORKDIR /src

RUN CGO_ENABLED=0 go install github.com/grpc-ecosystem/grpc-health-probe@v0.4.61

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/app ./cmd

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/app /app
COPY --from=builder /go/bin/grpc-health-probe /bin/grpc_health_probe

EXPOSE 50051 9090
ENTRYPOINT ["/app"]