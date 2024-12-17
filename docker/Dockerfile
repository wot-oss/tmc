FROM alpine:latest

ARG VERSION

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

# fetch the tmc by the given version for linux
COPY ./docker/download-cli.sh .
RUN export VERSION=$VERSION
RUN ./download-cli.sh; rm ./download-cli.sh
RUN chmod 755 ./tmc; mv ./tmc /usr/bin;
RUN tmc repo add --type file catalog $WORKDIR

# always run the tmc for this docker
CMD ["serve", "--host", "0.0.0.0" ,"--port",  "8080", "-d", "."]
ENTRYPOINT ["tmc"]
