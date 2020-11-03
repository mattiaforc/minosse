FROM alpine:3.12

WORKDIR /web
COPY minosse /usr/bin/
RUN ["chmod", "+x", "/usr/bin/minosse"]

CMD ["minosse"]