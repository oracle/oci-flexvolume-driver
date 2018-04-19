FROM oraclelinux:7.3

COPY dist/bin/oci /oci
COPY ./deploy.sh /deploy.sh

CMD ["/deploy.sh"]
