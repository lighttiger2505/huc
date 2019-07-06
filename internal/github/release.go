package github

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// https://developer.github.com/v4/object/release/
type Release struct {
	ID          githubv4.ID
	Name        githubv4.String
	TagName     githubv4.String
	Description githubv4.String
}

func (i *Release) ToString() string {
	return fmt.Sprintf("%d\nTitle: %s\n\n%s",
		i.ID,
		i.TagName,
		i.Description,
	)
}

type ListProjectReleaseOption struct {
	Num       int
	Sort      githubv4.ReleaseOrderField
	Direction githubv4.OrderDirection
}

func ListRelease(token, repositoryOwner, repositoryName string, opt *ListProjectReleaseOption) ([]Release, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	var q struct {
		Repository struct {
			DatabaseID githubv4.Int
			URL        githubv4.URI
			Releases   struct {
				Nodes []Release
			} `graphql:"releases(first:$releaseFirst, orderBy:$releaseOrder)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repositoryOwner),
		"repositoryName":  githubv4.String(repositoryName),
		"releaseOrder": githubv4.ReleaseOrder{
			Direction: opt.Direction,
			Field:     opt.Sort,
		},
		"releaseFirst": githubv4.Int(opt.Num),
	}

	if err := client.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}

	return q.Repository.Releases.Nodes, nil
}
