FROM scratch

MAINTAINER shrikantsharat.k@gmail.com

ADD bin/httpbun-docker /

ENV BIND=0.0.0.0:80
EXPOSE 80

CMD ["/httpbun-docker"]
