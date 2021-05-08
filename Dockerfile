FROM scratch

MAINTAINER shrikantsharat.k@gmail.com

ADD httpbun-docker /

EXPOSE 3090

CMD ["/httpbun-docker"]
