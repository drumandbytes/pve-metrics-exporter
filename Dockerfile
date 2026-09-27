# Static CGO-free binary; distroless/static brings CA certs and tzdata.
FROM golang:1.27-trixie AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /pve-metrics-exporter .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /pve-metrics-exporter /pve-metrics-exporter
EXPOSE 9221
ENTRYPOINT ["/pve-metrics-exporter"]
