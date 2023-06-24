FROM scratch

MAINTAINER shrikantsharat.k@gmail.com

ADD bin/httpbun-docker /

ENV PATH=""
EXPOSE 80

ENTRYPOINT ["/httpbun-docker", "--bind", "0.0.0.0:80"]
