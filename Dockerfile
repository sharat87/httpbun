FROM scratch

MAINTAINER shrikantsharat.k@gmail.com

ADD bin/httpbun-docker /

ENV HTTPBUN_BIND=0.0.0.0:80
EXPOSE 80

ENTRYPOINT ["/httpbun-docker"]
