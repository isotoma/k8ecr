1.3.0 (2018-03-20)
------------------

- Support in-cluster operation. If no kubeconfig file is found, it assumes it is running in cluster.
- Support automated deployment without presenting a choice. Provide an image name, or "-" for all images, on the command line for k8ecr deploy.
- Building with `CGO_ENABLED=0` to support operation on alpine
