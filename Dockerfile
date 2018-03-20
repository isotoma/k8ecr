FROM alpine
ENV AWS_REGION eu-west-2
RUN apk add --no-cache ca-certificates
ADD k8ecr /
CMD while true; do ./k8ecr deploy $NAMESPACE -; sleep 60; done
