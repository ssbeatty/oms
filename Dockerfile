FROM ubuntu:18.04
COPY ./configs/config.yaml.example /etc/oms/config.yaml
COPY ./release/oms_linux_amd64 /opt/oms/oms_linux_amd64

EXPOSE 8080
WORKDIR /opt/oms
CMD ["./oms_linux_amd64", "--config=/etc/oms/config.yaml"]