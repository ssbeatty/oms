FROM ubuntu:18.04
COPY internal/config/config.yaml.example /etc/oms/config.yaml
COPY internal/config/config.yaml.example /opt/oms/config.yaml.example
COPY ./release/oms_linux_amd64 /opt/oms/oms_linux_amd64

EXPOSE 8080
WORKDIR /opt/oms
CMD ["./oms_linux_amd64", "--config=/etc/oms/config.yaml"]