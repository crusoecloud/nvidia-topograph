FROM --platform=$BUILDPLATFORM golang:1.24.7 AS builder

WORKDIR /go/src/github.com/NVIDIA/topograph
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN make build-${TARGETOS}-${TARGETARCH}

# Runtime base pinned to an Alpine whose ca-certificates-bundle no longer
# ships the roots distrusted after the eMudhra CA compromise
# (CVE-2026-111943): 3.24.3 / ca-certificates-bundle 20261020-r0. The
# unpinned alpine:3 used until 0.1.0-cmk.3 carried emSign ECC Root CA - G3.
# Multi-arch index digest, so it resolves for amd64 and arm64 alike.
FROM alpine:3.24.3@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS runtime

COPY --from=builder /go/src/github.com/NVIDIA/topograph/bin/* /usr/local/bin/

# Fail the build if the trust store still carries a distrusted root, so a
# future base bump cannot silently reintroduce one.
FROM runtime AS verify
RUN apk add --no-cache openssl >/dev/null \
    && openssl crl2pkcs7 -nocrl -certfile /etc/ssl/certs/ca-certificates.crt \
       | openssl pkcs7 -print_certs -noout | grep '^subject=' > /tmp/roots.txt \
    && ! grep -E 'CN ?= ?(emSign ECC Root CA - G3|AAA Certificate Services|emSign Root TLS CA - G3|emSign Root CS CA - G3)' /tmp/roots.txt \
    && echo "trust store clean: $(wc -l < /tmp/roots.txt) roots"

FROM runtime
# Depends on the verify stage so it always runs; the root list it leaves
# behind doubles as a record of what the image trusts.
COPY --from=verify /tmp/roots.txt /usr/share/doc/trust-store-roots.txt
