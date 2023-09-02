FROM scratch

MAINTAINER shrikantsharat.k@gmail.com

ARG TARGETARCH
ADD bin/httpbun-docker-$TARGETARCH /httpbun

ENV PATH="___httpbun_unset_marker"
ENV HOME="___httpbun_unset_marker"
ENV HOSTNAME="___httpbun_unset_marker"

EXPOSE 80

ENTRYPOINT ["/httpbun", "--bind", "0.0.0.0:80"]
