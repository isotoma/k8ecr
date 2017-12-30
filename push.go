package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"encoding/base64"

	docker "docker.io/go-docker"
	"docker.io/go-docker/api/types"
	"github.com/aws/aws-sdk-go/service/ecr"
)

type PushCommand struct{}

var pushCommand PushCommand

func getCredentials() (types.AuthConfig, error) {
	svc := ecr.New(createSession())
	response, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return types.AuthConfig{}, err
	}
	token, err := base64.StdEncoding.DecodeString(*response.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return types.AuthConfig{}, err
	}
	parts := strings.Split(string(token), ":")
	username := parts[0]
	password := parts[1]
	endpoint := *response.AuthorizationData[0].ProxyEndpoint
	return types.AuthConfig{
		Username:      username,
		Password:      password,
		ServerAddress: endpoint,
	}, nil
}

func login() (*docker.Client, string, error) {
	creds, err := getCredentials()
	if err != nil {
		return nil, "", err
	}
	cli, err := docker.NewEnvClient()
	if err != nil {
		return cli, "", err
	}
	_, err = cli.RegistryLogin(context.Background(), creds)
	if err != nil {
		return nil, "", err
	}
	return cli, creds.ServerAddress[8:], nil
}

func push(client *docker.Client, endpoint string, repo string, version string) error {
	source := fmt.Sprintf("%s:%s", repo, version)
	target := fmt.Sprintf("%s:%s", endpoint, version)
	return client.ImageTag(context.Background(), source, target)
}

func pushRepository(name string, versions []string) error {
	cli, endpoint, err := login()
	if err != nil {
		return err
	}
	for _, v := range versions {
		err := push(cli, endpoint, name, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (x *PushCommand) Execute(args []string) error {
	if len(args) < 2 {
		return errors.New("push REPOSITORY VERSION...")
	}
	return pushRepository(args[0], args[1:])

}

func init() {
	parser.AddCommand("push",
		"Push",
		"Push an image to ECR",
		&pushCommand)
}
