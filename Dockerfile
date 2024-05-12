from docker.io/library/golang:1.22-alpine as build
WORKDIR /agent
COPY . .
RUN go mod vendor && \
        CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o /dns-sync .

FROM scratch
COPY --from=build /dns-sync /dns-sync
LABEL org.opencontainers.image.source https://github.com/the-maldridge/mkt-nomad-dns
ENTRYPOINT ["/dns-sync"]
