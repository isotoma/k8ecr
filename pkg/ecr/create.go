package ecr

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/iam"
)

// Get the context name for the current cluster
// This is used to find the AWS roles for the cluster
// This is how kops configures things specifically
func getContext() string {
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	context := strings.TrimSpace(string(output))
	return context
}
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

// PrincipalEntry in an IAM Policy
type PrincipalEntry struct {
	AWS []string
}

// StatementEntry in an IAM Policy
type StatementEntry struct {
	Sid       string
	Effect    string
	Principal PrincipalEntry
	Action    []string
}

// PolicyDocument in an IAM Policy
type PolicyDocument struct {
	Version   string
	Statement []StatementEntry
}

// CreateRepository creates the named repository
func (r *Registry) CreateRepository(name string) (*ecr.Repository, error) {
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
					"ecr:DescribeImages",
				},
			},
		},
	}

	b, err := json.Marshal(&policy)
	if err != nil {
		return nil, err
	}

	resp, createErr := r.service.CreateRepository(&ecr.CreateRepositoryInput{
		RepositoryName: &name,
	})

	if createErr != nil {
		return nil, createErr
	}

	var repo = resp.Repository
	r.service.SetRepositoryPolicy(&ecr.SetRepositoryPolicyInput{
		RepositoryName: aws.String(name),
		PolicyText:     aws.String(string(b)),
	})
	return repo, nil
}
