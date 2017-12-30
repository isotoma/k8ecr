package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/iam"
)

type CreateCommand struct{}

var createCommand CreateCommand

func getClusterRole(cluster string, role string) string {
	svc := iam.New(createSession())
	name := fmt.Sprintf("%s.%s", role, cluster)
	input := &iam.GetRoleInput{
		RoleName: &name,
	}
	result, err := svc.GetRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				fmt.Println(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				fmt.Println(iam.ErrCodeServiceFailureException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return ""
	}
	return *result.Role.Arn
}

func getPrincipals(cluster string) []string {
	masters := getClusterRole(cluster, "masters")
	nodes := getClusterRole(cluster, "nodes")
	return []string{masters, nodes}
}

type PrincipalEntry struct {
	AWS []string
}

type StatementEntry struct {
	Sid       string
	Effect    string
	Principal PrincipalEntry
	Action    []string
}

type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

func createRepository(name string) error {
	context := getContext()
	principals := getPrincipals(context)
	policy := PolicyDocument{
		Version: "2008-10-17",
		Statement: []StatementEntry{
			StatementEntry{
				Sid:    "Cluster access",
				Effect: "Allow",
				Principal: PrincipalEntry{
					AWS: principals,
				},
				Action: []string{
					"ecr:GetDownloadUrlForLayer",
					"ecr:BatchGetImage",
					"ecr:BatchCheckLayerAvailability",
				},
			},
		},
	}
	b, err := json.Marshal(&policy)
	if err != nil {
		return err
	}

	svc := ecr.New(createSession())
	svc.CreateRepository(&ecr.CreateRepositoryInput{
		RepositoryName: &name,
	})
	svc.SetRepositoryPolicy(&ecr.SetRepositoryPolicyInput{
		RepositoryName: aws.String(name),
		PolicyText:     aws.String(string(b)),
	})
	return nil
}

func (x *CreateCommand) Execute(args []string) error {
	if len(args) == 0 {
		return errors.New("No repository name specified")
	} else {
		createRepository(args[0])
	}
	return nil
}

func init() {
	parser.AddCommand("create",
		"Create",
		"Create an ECR repository and grant read permissions to your cluster",
		&createCommand)
}
