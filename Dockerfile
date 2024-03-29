FROM umputun/baseimage:buildgo-latest as build

ARG GIT_BRANCH
ARG GITHUB_SHA
ARG CI

ENV GOFLAGS="-mod=vendor"

ADD . /build
WORKDIR /build

RUN \
    if [ -z "$CI" ] ; then \
    echo "runs outside of CI" && version=$(git rev-parse --abbrev-ref HEAD)-$(git log -1 --format=%h)-$(date +%Y%m%dT%H:%M:%S); \
    else version=${GIT_BRANCH}-${GITHUB_SHA:0:7}-$(date +%Y%m%dT%H:%M:%S); fi && \
    echo "version=$version" && \
    cd app && go build -o /build/telegram-rt-bot -ldflags "-X main.revision=${version} -s -w"


FROM umputun/baseimage:app-latest
COPY --from=build /build/telegram-rt-bot /srv/telegram-rt-bot
COPY data/*.data /srv/data/
COPY data/logs.html /srv/logs.html

RUN chown -R app:app /srv

EXPOSE 18001
WORKDIR /srv
CMD ["/srv/telegram-rt-bot"]
