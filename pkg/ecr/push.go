package ecr

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
		ServerAddress: endpoint[8:], // strip the https://
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
	fmt.Println("Logging into", creds.ServerAddress)
	response, err := cli.RegistryLogin(context.Background(), creds)
	if err != nil {
		return nil, creds, err
	}
	fmt.Println(response)
	return cli, creds, nil
}

func registryAuth(creds types.AuthConfig) string {
	// conveniently types.AuthConfig has terms that we need for the
	// authorisation header
	b, err := json.Marshal(&creds)
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func tag(client *docker.Client, endpoint string, repo string, version string) error {
	source := fmt.Sprintf("%s:%s", repo, version)
	target := fmt.Sprintf("%s/%s:%s", endpoint, repo, version)
	fmt.Println("Tagging", source, target)
	return client.ImageTag(context.Background(), source, target)
}

func startPush(client *docker.Client, creds types.AuthConfig, repo string, version string) (io.ReadCloser, error) {
	err := tag(client, creds.ServerAddress, repo, version)
	if err != nil {
		return nil, err
	}
	image := fmt.Sprintf("%s/%s:%s", creds.ServerAddress, repo, version)
	fmt.Println("\n\nPushing ", image)
	stream, err := client.ImagePush(context.Background(),
		image,
		types.ImagePushOptions{
			RegistryAuth: registryAuth(creds),
		})
	if err != nil {
		return nil, err
	}
	return stream, nil

}

func getNextLine(stream *bufio.Reader) (*ProgressLine, error) {
	b, err := stream.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	data := ProgressLine{}
	jsonErr := json.Unmarshal(b, &data)
	if jsonErr != nil {
		return nil, jsonErr
	}
	if data.Error != "" {
		return nil, errors.New(data.Error)
	}
	return &data, nil
}

func formatProgress(data *ProgressLine) string {
	progress := data.Progress
	if progress == "" {
		progress = data.Status
	}
	return fmt.Sprintf("%s: %s", data.ID, progress)
}

type ProgressLine struct {
	ID             string
	Status         string
	Progress       string
	ProgressDetail map[string]int
	Error          string
}

type ProgressDisplay struct {
	Bars  map[string]int
	Lines int
}

func updateLine(lineno int, message string) {
	escape := "\x1b"
	fmt.Printf("%s[1000D", escape)       // Move left
	fmt.Printf("%s[%dA", escape, lineno) // Move up
	fmt.Printf("%s", message)
	fmt.Printf("%s[%dB", escape, lineno) // Move down
}

// Update the progress and print the output
func (p *ProgressDisplay) Update(data *ProgressLine) {
	if data.ID != "" {
		line, ok := p.Bars[data.ID]
		if ok {
			updateLine(p.Lines-line, formatProgress(data))
		} else {
			p.Bars[data.ID] = p.Lines
			p.Lines++
			fmt.Println(data.ID)
		}
	}
}

func push(client *docker.Client, creds types.AuthConfig, repo string, version string) error {
	rawStream, err := startPush(client, creds, repo, version)
	if err != nil {
		return err
	}
	stream := bufio.NewReader(rawStream)
	display := ProgressDisplay{
		Bars:  make(map[string]int),
		Lines: 0,
	}
	for {
		data, err := getNextLine(stream)
		if err != io.EOF && err != nil {
			rawStream.Close()
			return err
		}
		if data != nil {
			if data.Error != "" {
				fmt.Println("\n\n", data.Error)
				return errors.New(data.Error)
			}
			display.Update(data)
		}
		if err == io.EOF {
			fmt.Println("\n\nDone")
			rawStream.Close()
			return nil
		}
	}
}

// PushRepository pushes
func (r *Registry) PushRepository(name string, versions []string) error {
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
