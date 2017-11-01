IMAGE=wcr.io/owainlewis/oci-flexvolume-driver
TAG=0.2.0

docker build -t $IMAGE:$TAG .
docker push $IMAGE:$TAG
