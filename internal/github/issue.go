package github

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Query some details about a repository, an issue in it, and its comments.
type GithubV4Actor struct {
	Login     githubv4.String
	AvatarURL githubv4.URI `graphql:"avatarUrl(size:72)"`
	URL       githubv4.URI
}

type Issue struct {
	ID              githubv4.ID
	Number          githubv4.Int
	Author          GithubV4Actor
	PublishedAt     githubv4.DateTime
	LastEditedAt    *githubv4.DateTime
	Editor          *GithubV4Actor
	Title           githubv4.String
	Body            githubv4.String
	ViewerCanUpdate githubv4.Boolean
}

func (i *Issue) ToString() string {
	return fmt.Sprintf("Issue Number: %d (%s)\nTitle: %s\n\n%s",
		i.Number,
		i.ID,
		i.Title,
		i.Body,
	)
}

type ListProjectIssueOption struct {
	Num       int
	Sort      githubv4.IssueOrderField
	Direction githubv4.OrderDirection
	States    githubv4.IssueState
	Labels    []string
}

func ListIssue(token, repositoryOwner, repositoryName string, opt *ListProjectIssueOption) ([]Issue, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	var q struct {
		Repository struct {
			DatabaseID githubv4.Int
			URL        githubv4.URI

			Issues struct {
				Nodes []Issue
			} `graphql:"issues(first:$issueFirst, states:$issueStates, orderBy:$issueOrder)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repositoryOwner),
		"repositoryName":  githubv4.String(repositoryName),
		"issueOrder": githubv4.IssueOrder{
			Direction: opt.Direction,
			Field:     opt.Sort,
		},
		"issueStates": []githubv4.IssueState{opt.States},
		"issueFirst":  githubv4.Int(opt.Num),
	}

	if err := client.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}

	return q.Repository.Issues.Nodes, nil
}
