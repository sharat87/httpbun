FROM golang AS builder
WORKDIR /w/
COPY ./ /w/
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod/cache go mod tidy && go build -a -installsuffix cgo -o /httpbun .

FROM scratch

LABEL org.opencontainers.image.authors="shrikantsharat.k@gmail.com"

COPY --from=builder /httpbun /httpbun

ENV PATH="___httpbun_unset_marker"
ENV HOME="___httpbun_unset_marker"
ENV HOSTNAME="___httpbun_unset_marker"

EXPOSE 80
EXPOSE 443

ENTRYPOINT ["/httpbun"]
