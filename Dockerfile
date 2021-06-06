FROM scratch

MAINTAINER shrikantsharat.k@gmail.com

ADD bin/httpbun-docker /

ENV HOST=0.0.0.0
ENV PORT=80
EXPOSE 80

CMD ["/httpbun-docker"]
