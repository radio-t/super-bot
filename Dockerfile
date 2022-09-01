FROM umputun/baseimage:buildgo-latest as build

ADD . /build
WORKDIR  /build
RUN go test ./app/...
RUN cd app && CGO_ENABLED=0 GOOS=linux go build -o /target/telegram-rt-bot -ldflags \
    "-X main.revision=$(git rev-parse --abbrev-ref HEAD)-$(git describe --abbrev=7 --always --tags)-$(date +%Y%m%d-%H:%M:%S)"


FROM umputun/baseimage:app
COPY --from=build /target/telegram-rt-bot /srv/telegram-rt-bot
COPY data/*.data /srv/data/
COPY data/logs.html /srv/logs.html

RUN chown -R app:app /srv

EXPOSE 18001
WORKDIR /srv
CMD ["/srv/telegram-rt-bot"]
