FROM ubuntu:18.04
COPY internal/config/config.yaml.docker /etc/oms/config.yaml
COPY internal/config/config.yaml.docker /opt/oms/config.yaml.example
COPY ./release/oms_linux_amd64 /opt/oms/oms_linux_amd64
COPY ./entrypoint.sh /opt/oms/entrypoint.sh
RUN chmod +x /opt/oms/entrypoint.sh

WORKDIR /opt/oms
ENTRYPOINT ["/opt/oms/entrypoint.sh"]
CMD ["/opt/oms/oms_linux_amd64", "--config=/etc/oms/config.yaml"]