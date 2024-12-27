FROM scratch

ARG TARGETARCH
ADD bin/httpbun-docker-$TARGETARCH /httpbun

ENV PATH="___httpbun_unset_marker"
ENV HOME="___httpbun_unset_marker"
ENV HOSTNAME="___httpbun_unset_marker"

EXPOSE 80
EXPOSE 443

ENTRYPOINT ["/httpbun"]
