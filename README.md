# k8ecr

Utility for managing ecr repositories with kubernetes

## Building

    make vendor
    make

## Concepts

k8ecr provides tooling to make it easier to use docker images from ECR repositories in your Kubernetes clusters created by kops. 

It can:

- create ECR repositories and grant appropriate permissions to your cluster roles.
- push images to ECR repositories directly.
- issue appropriate kubectl set image commands to update deployments and associated resources.

## Usage

    k8ecr create REPOSITORY
    k8ecr push REPOSITORY VERSION...
    k8ecr deploy NAMESPACE

## Environment variables

k8ecr expects KUBECONFIG and AWS_PROFILE to be correctly configured for your environment.

It uses these to interact with your cluster and your AWS account.

## Creating repositories

    k8ecr create REPOSITORY

This will create an ECR repository in the current profile, and grant:

    ecr:GetDownloadUrlForLayer
    ecr:BatchGetImage
    ecr:BatchCheckLayerAvailability
    ecr:DescribeImages

To the IAM master and nodes role for the current cluster. These permissions will allow
deployments to operate successfully.

## Pushing images

    k8ecr push REPOSITORY VERSION...

This will log in to ECR, then push images to the remote repository of the same name with the specified versions.  For example:

    k8ecr push myimage 1.0.0 latest

Will push 1.0.0 and latest tags.

## Deploying

    k8ecr deploy [NAMESPACE]

This will compare all deployments and the must recent version numbers available and present options for deploying images.

All possible upgrade options for the specified namespace are shown.
