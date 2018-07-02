#!/bin/bash

CMD='curl -fL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.4.1/dep-linux-amd64 &&
chmod +x /usr/local/bin/dep &&
cd /app/src/k8ecr &&
dep ensure &&
make &&
cp k8ecr /build'

docker run --rm -it \
       -e GOPATH=/app \
       -v $(pwd)/build:/build \
       -v $(pwd):/app/src/k8ecr \
       golang sh -c "$CMD"
