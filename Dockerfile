FROM umputun/baseimage:buildgo-latest as build

ADD . /build
WORKDIR  /build
RUN go test ./app/...
RUN cd app && CGO_ENABLED=0 GOOS=linux go build -o /target/gitter-rt-bot -ldflags \
 "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S)"


# Run
FROM umputun/baseimage:app
COPY --from=build /target/gitter-rt-bot /srv/gitter-rt-bot
COPY data/ /srv/
RUN chown -R app:app /srv

EXPOSE 18001
WORKDIR /srv
CMD ["/srv/gitter-rt-bot"]
