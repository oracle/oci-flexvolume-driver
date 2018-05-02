FROM iad.ocir.io/oracle/oci-flexvolume-driver-system-test:1.0.2

COPY dist /dist
COPY test/system /test/system

WORKDIR /test/system

CMD ["./runner.py"]
