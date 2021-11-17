FROM ubuntu:18.04
COPY ./configs/config.yaml.example ./config.yaml
COPY ./release/oms_linux_amd64 .
CMD ["./oms_linux_amd64"]