package github

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

type PullRequest struct {
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

func (i *PullRequest) ToString() string {
	return fmt.Sprintf("Pull Request Number: %d (%s)\nTitle: %s\n\n%s",
		i.Number,
		i.ID,
		i.Title,
		i.Body,
	)
}

type ListProjectPullRequestOption struct {
	Num       int
	Sort      githubv4.IssueOrderField
	Direction githubv4.OrderDirection
	States    githubv4.PullRequestState
	Labels    []string
}

func ShowPullRequest(token, repositoryOwner, repositoryName string, number int) (*PullRequest, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	// Target object pullRequests https://developer.github.com/v4/object/repository/
	var q struct {
		Repository struct {
			DatabaseID  githubv4.Int
			URL         githubv4.URI
			PullRequest PullRequest `graphql:"pullRequest(number:$pullRequestNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	variables := map[string]interface{}{
		"repositoryOwner":   githubv4.String(repositoryOwner),
		"repositoryName":    githubv4.String(repositoryName),
		"pullRequestNumber": githubv4.Int(number),
	}

	if err := client.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}

	return &q.Repository.PullRequest, nil
}

func ListPullRequest(token, repositoryOwner, repositoryName string, opt *ListProjectPullRequestOption) ([]PullRequest, error) {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	// Target object pullRequests https://developer.github.com/v4/object/repository/
	var q struct {
		Repository struct {
			DatabaseID githubv4.Int
			URL        githubv4.URI

			PullRequests struct {
				Nodes []PullRequest
			} `graphql:"pullRequests(first:$pullRequestFirst, states:$pullRequestState, orderBy:$pullRequestOrder)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String(repositoryOwner),
		"repositoryName":  githubv4.String(repositoryName),
		"pullRequestOrder": githubv4.IssueOrder{
			Direction: opt.Direction,
			Field:     opt.Sort,
		},
		"pullRequestState": []githubv4.PullRequestState{opt.States},
		"pullRequestFirst": githubv4.Int(50),
	}

	if err := client.Query(context.Background(), &q, variables); err != nil {
		return nil, err
	}

	return q.Repository.PullRequests.Nodes, nil
}
