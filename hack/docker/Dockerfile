FROM appscode/base:8.6

ENV PGROOT /var/db

FROM appscode/base:8.6

RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates netcat \
  && rm -rf /var/lib/apt/lists/* /usr/share/doc /usr/share/man /tmp/*

COPY gearmand /gearmand
COPY gcron /gcron

COPY sv /etc/sv/
RUN ln -s /etc/sv /etc/service

COPY runit.sh /runit.sh
ENTRYPOINT ["/runit.sh"]
EXPOSE 4730
