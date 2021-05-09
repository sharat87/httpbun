FROM scratch

MAINTAINER shrikantsharat.k@gmail.com

ADD bin/httpbun-docker /

ENV PORT=80
EXPOSE 80

CMD ["/httpbun-docker"]
