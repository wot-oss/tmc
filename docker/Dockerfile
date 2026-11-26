FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
COPY cmd ./cmd
COPY internal ./internal

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-X github.com/wot-oss/tmc/internal/utils.TmcVersion=${VERSION}" \
    -o /out/tmc .

FROM alpine:latest

ARG VERSION=dev

# externally configurable environment variables
# either set default here or set with docker run option -e
ENV TMC_URLCONTEXTROOT=
ENV TMC_LOGLEVEL=INFO
ENV TMC_CORSALLOWEDORIGINS=
ENV TMC_CORSALLOWEDHEADERS=
ENV TMC_CORSALLOWCREDENTIALS=
ENV TMC_CORSMAXAGE=86500

LABEL name="W3C Thing Model Catalog" \
      maintainer="https://github.com/wot-oss" \
      version=${VERSION} \
      summary="A catalog service to host and share W3C thing models"
      
ENV WORKDIR /thingmodels

RUN apk update \
 && apk add --no-cache curl jq \
 && rm -rf /var/cache/apk/*

WORKDIR $WORKDIR
COPY --from=build /out/tmc /usr/bin/tmc

# always run the tmc for this docker
CMD ["serve", "--host", "0.0.0.0" ,"--port",  "8080", "-d", "."]
ENTRYPOINT ["tmc"]
