FROM alpine:latest

# externally configurable environment variables
# either set default here or set with docker run option -e
ENV TMC_URLCONTEXTROOT=
ENV TMC_LOGLEVEL=INFO
ENV TMC_CORSALLOWEDORIGINS=
ENV TMC_CORSALLOWEDHEADERS=
ENV TMC_CORSALLOWCREDENTIALS=
ENV TMC_CORSMAXAGE=86500

LABEL name="W3C Thing Model Catalog" \
      maintainer="https://github.com/web-of-things-open-source" \
      version=${VERSION} \
      summary="A catalog service to host and share W3C thing models"
      
ENV WORKDIR /thingmodels

RUN apk update \
 && apk add --no-cache curl jq \
 && rm -rf /var/cache/apk/*

WORKDIR $WORKDIR 

# fetch the latest tm-catalog-cli for linux
COPY ./docker/download-cli.sh .
RUN ./download-cli.sh; rm ./download-cli.sh
RUN chmod 755 ./tm-catalog-cli; mv ./tm-catalog-cli /usr/bin; 
RUN tm-catalog-cli remote add --type file catalog $WORKDIR

# always run the tm-catalog-cli for this docker
CMD ["serve", "--host", "0.0.0.0" ,"--port",  "8080", "-d", "."]
ENTRYPOINT ["tm-catalog-cli"]
