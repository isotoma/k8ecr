package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"encoding/base64"
	"encoding/json"

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

func login() (*docker.Client, types.AuthConfig, error) {
	creds, err := getCredentials()
	if err != nil {
		return nil, creds, err
	}
	cli, err := docker.NewEnvClient()
	if err != nil {
		return cli, creds, err
	}
	_, err = cli.RegistryLogin(context.Background(), creds)
	if err != nil {
		return nil, creds, err
	}
	return cli, creds, nil
}

func registryAuth(creds types.AuthConfig) string {
	b := fmt.Sprintf("%s:%s", creds.Username, creds.Password)
	return base64.StdEncoding.EncodeToString([]byte(b))
}

func tag(client *docker.Client, endpoint string, repo string, version string) error {
	source := fmt.Sprintf("%s:%s", repo, version)
	target := fmt.Sprintf("%s:%s", endpoint, version)
	return client.ImageTag(context.Background(), source, target)
}

func push(client *docker.Client, creds types.AuthConfig, repo string, version string) error {
	err := tag(client, creds.ServerAddress[8:], repo, version)
	if err != nil {
		return err
	}
	image := fmt.Sprintf("%s/%s", creds.ServerAddress[8:], repo)
	stream, err := client.ImagePush(context.Background(),
		image,
		types.ImagePushOptions{
			RegistryAuth: registryAuth(creds),
		})
	if err != nil {
		return err
	}
	buffered := bufio.NewReader(stream)
	for {
		b, err := buffered.ReadBytes('\n')
		if err != nil {
			stream.Close()
			return err
		}
		data := make(map[string]interface{})
		jsonErr := json.Unmarshal(b, &data)
		if jsonErr != nil {
			stream.Close()
			return jsonErr
		}
		fmt.Println(data)
		if err == io.EOF {
			fmt.Println("\nDone")
			stream.Close()
			return nil
		}
	}
}

func pushRepository(name string, versions []string) error {
	cli, creds, err := login()
	if err != nil {
		return err
	}
	for _, v := range versions {
		err := push(cli, creds, name, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// Execute the push command
func (x *PushCommand) Execute(args []string) error {
	if len(args) < 2 {
		return errors.New("push REPOSITORY VERSION")
	}
	return pushRepository(args[0], args[1:])

}

func init() {
	parser.AddCommand("push",
		"Push",
		"Push an image to ECR",
		&pushCommand)
}
