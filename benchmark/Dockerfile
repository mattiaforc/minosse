FROM debian

WORKDIR /web
COPY minosse /usr/bin/
COPY public /web/public
COPY config /web/config
RUN ["chmod", "+x", "/usr/bin/minosse"]

CMD ["minosse"]