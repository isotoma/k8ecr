package ecr

import (
	"github.com/aws/aws-sdk-go/service/ecr"
)

// GetAllRepositories Get all the repositories in the registry
func getAllRepositories(svc *ecr.ECR) ([]string, error) {
	response, err := svc.DescribeRepositories(&ecr.DescribeRepositoriesInput{})
	if err != nil {
		return nil, err
	}
	repositories := make([]string, len(response.Repositories))
	for i, r := range response.Repositories {
		repositories[i] = *r.RepositoryName
	}
	return repositories, nil
}

func getTagsForRepositoryPage(svc *ecr.ECR, repository string, tagList []string, nextToken *string) ([]string, *string, error) {
	response, err := svc.DescribeImages(&ecr.DescribeImagesInput{
		RepositoryName: &repository,
		NextToken:      nextToken,
	})
	if err != nil {
		return tagList, nil, err
	}
	for _, i := range response.ImageDetails {
		for _, t := range i.ImageTags {
			if *t != "latest" {
				tagList = append(tagList, *t)
			}
		}
	}
	return tagList, response.NextToken, nil
}

// GetTagsForRepository gets all the tags in a specified repository
func getTagsForRepository(svc *ecr.ECR, repository string) ([]string, error) {
	tagList := make([]string, 0)
	tagList, nextToken, err := getTagsForRepositoryPage(svc, repository, tagList, nil)
	if err != nil {
		return nil, err
	}
	for nextToken != nil {
		tagList, nextToken, err = getTagsForRepositoryPage(svc, repository, tagList, nextToken)
		if err != nil {
			return nil, err
		}

	}
	return tagList, nil
}
