package ecr

import (
	"github.com/aws/aws-sdk-go/service/ecr"
)

// Repository represents a repository within the registry
type Repository struct {
	Name      string
	LatestTag string
	Tags      []string
}

// Registry represents your an ECR in a region
type Registry struct {
	service      *ecr.ECR
	Repositories map[string]Repository
}

// NewRegistry creates a new Registry object
func NewRegistry() *Registry {
	return &Registry{
		service:      ecr.New(createSession()),
		Repositories: make(map[string]Repository),
	}
}

// FetchAll gets all the repositories and updates their tags and latest
func (r *Registry) FetchAll() error {
	repositories, err := getAllRepositories(r.service)
	if err != nil {
		return err
	}
	for _, repo := range repositories {
		tags, err := getTagsForRepository(r.service, repo)
		if err != nil {
			return err
		}
		latest := latestVersion(tags)
		r.Repositories[repo] = Repository{
			Name:      repo,
			LatestTag: latest,
			Tags:      tags,
		}
	}
	return nil
}
