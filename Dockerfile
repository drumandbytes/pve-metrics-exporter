# Two-stage build. Unlike a wrapped vendor binary (see e.g.
# drumandbytes/nordvpn), this is our own pure Go code with CGO disabled -
# no .deb, no shared libraries to hunt down with ldd, just a single static
# binary. distroless/static (not /base) is enough: no libc needed at all,
# and it already bundles CA certificates + tzdata, which is all the
# runtime needs (HTTPS to the Proxmox API; TZ isn't currently used but
# costs nothing to have available).
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
