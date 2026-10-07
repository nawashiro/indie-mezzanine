# syntax=docker/dockerfile:1
FROM golang:1.27.1@sha256:162be5298a40ed317005c8339c6de4d10d3eef336d66dc8e9259b03ab9d3a6d2 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY cmd ./cmd
COPY LICENSE /out/LICENSE
COPY docs/LICENSING.md /out/LICENSING.md
COPY docs/THIRD_PARTY_NOTICES.txt /out/THIRD_PARTY_NOTICES.txt
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mezzanine ./cmd/mezzanine \
    && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=build /out/mezzanine /mezzanine
COPY --from=build /out/LICENSE /LICENSE
COPY --from=build /out/LICENSING.md /LICENSING.md
COPY --from=build /out/THIRD_PARTY_NOTICES.txt /THIRD_PARTY_NOTICES.txt
COPY --from=build --chown=65532:65532 /out/data /data
USER 65532:65532
ENV DATA_DIR=/data LISTEN_ADDR=:8080
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD ["/mezzanine", "healthcheck"]
ENTRYPOINT ["/mezzanine"]
