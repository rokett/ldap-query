FROM golang:alpine as builder

ENV VERSION="1.0.0"

WORKDIR $GOPATH/src/github.com/rokett
RUN \
    apk add --no-cache git && \
    git clone --branch $VERSION --depth 1 https://github.com/rokett/ldap-query.git ldap-query && \
    cd ldap-query && \
    BUILD=$(git rev-list -1 HEAD) && \
    CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-X main.version=$VERSION -X main.build=$BUILD -s -w -extldflags '-static'" -o ldap-query

FROM scratch
LABEL maintainer="rokett@rokett.me"
COPY --from=builder go/src/github.com/rokett/ldap-query/ldap-query /

# Dynamic expose port
EXPOSE 9999

# TODO Arguments
ENTRYPOINT ["./ldap-queryd"]
