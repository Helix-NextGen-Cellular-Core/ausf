# Builds the Helix image ghcr.io/helix-nextgen-cellular-core/ausf:v1.0.0
# the same way it was built and verified:
#   docker build -t ausf:local .
FROM golang:1.16.0-stretch AS builder
WORKDIR /go/src/ausf
COPY . .
RUN go mod vendor \
 && make all && mkdir -p /out/free5gc/ausf && cp bin/* /out/free5gc/ausf/

FROM alpine:3.16
LABEL org.opencontainers.image.source="https://github.com/Helix-NextGen-Cellular-Core/ausf" \
      org.opencontainers.image.licenses="Apache-2.0"
RUN apk update && apk add -U vim strace net-tools curl netcat-openbsd bind-tools
COPY --from=builder /out/ /
WORKDIR /free5gc/ausf
