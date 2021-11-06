FROM golang AS mybuildstage
RUN go env -w GOPROXY=https://goproxy.cn && go env -w GO111MODULE=on
RUN go install github.com/gobuffalo/packr/packr@latest
COPY ./ /workdir
WORKDIR /workdir
RUN packr build -o oms cmd/omsd/main.go
FROM ubuntu:16.04
COPY ./configs/config.yaml.example ./config.yaml
COPY --from=mybuildstage /workdir/oms .
CMD ["./oms"]