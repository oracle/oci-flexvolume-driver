FROM busybox

MAINTAINER Owain Lewis <owain.lewis@oracle.com>

ADD dist/bin/oci /

ADD hack/driver.sh /

CMD ["./driver.sh"]
