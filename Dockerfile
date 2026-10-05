FROM scratch

ARG TARGETARCH
ADD bin/httpbun-docker-$TARGETARCH /httpbun

EXPOSE 80
EXPOSE 443

ENTRYPOINT ["/httpbun"]
