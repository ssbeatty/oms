FROM golang AS mybuildstage
COPY ./ /workdir
WORKDIR /workdir
RUN go env -w GOPROXY=https://goproxy.cn && go env -w GO111MODULE=on
RUN go get -u github.com/gobuffalo/packr/packr
RUN packr build -o oms -mod=mod
FROM ubuntu:16.04
COPY ./conf/config.yaml.example ./config.yaml
COPY --from=mybuildstage /workdir/oms .
CMD ["./oms"]