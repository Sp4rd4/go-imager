FROM golang:alpine AS build
WORKDIR /go/src/github.com/sp4rd4/go-imager
RUN apk add -U git \
	&& rm -rf /var/cache/apk/* \
	&& go get -u github.com/golang/dep/cmd/dep
COPY Gopkg.lock Gopkg.toml ./
RUN dep ensure --vendor-only
COPY . .

FROM build as compile-imgr
RUN go build ./cmd/imgr/main.go && mv main imgr

FROM build as compile-auth
RUN go build ./cmd/auth/main.go && mv main auth

FROM alpine as service
WORKDIR /app
CMD ./app

FROM service as imgr
COPY --from=compile-imgr /go/src/github.com/sp4rd4/go-imager/imgr ./app
COPY --from=compile-imgr /go/src/github.com/sp4rd4/go-imager/service/imgr/migrations ./migrations

FROM service as auth
COPY --from=compile-auth /go/src/github.com/sp4rd4/go-imager/auth ./app
COPY --from=compile-auth /go/src/github.com/sp4rd4/go-imager/service/auth/migrations ./migrations