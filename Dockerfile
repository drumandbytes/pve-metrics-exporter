# Static CGO-free binary; distroless/static brings CA certs and tzdata.
# Cross-compile on the runner's own arch; building arm64 under QEMU took minutes.
FROM --platform=$BUILDPLATFORM golang:1.27-trixie AS builder
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /pve-metrics-exporter .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /pve-metrics-exporter /pve-metrics-exporter
EXPOSE 9221
ENTRYPOINT ["/pve-metrics-exporter"]
