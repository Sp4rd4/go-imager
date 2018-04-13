FROM golang:alpine AS build
WORKDIR /go/src/github.com/sp4rd4/go-imager
RUN apk add -U git \
	&& rm -rf /var/cache/apk/* \
	&& go get -u github.com/golang/dep/cmd/dep
COPY Gopkg.lock Gopkg.toml ./
RUN dep ensure --vendor-only
COPY . .

FROM build as compile-images
RUN go build ./cmd/images/main.go && mv main images

FROM build as compile-auth
RUN go build ./cmd/auth/main.go && mv main auth

FROM alpine as service
WORKDIR /app
CMD ./app

FROM service as images
COPY --from=compile-images /go/src/github.com/sp4rd4/go-imager/images ./app
COPY --from=compile-images /go/src/github.com/sp4rd4/go-imager/service/images/migrations ./migrations

FROM service as auth
COPY --from=compile-auth /go/src/github.com/sp4rd4/go-imager/auth ./app
COPY --from=compile-auth /go/src/github.com/sp4rd4/go-imager/service/auth/migrations ./migrations