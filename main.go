package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Println("Invalid request", err)
	}
}

func run() error {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_GRAPHQL_TEST_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	// Query some details about a repository, an issue in it, and its comments.
	type githubV4Actor struct {
		Login     githubv4.String
		AvatarURL githubv4.URI `graphql:"avatarUrl(size:72)"`
		URL       githubv4.URI
	}

	var q struct {
		Repository struct {
			DatabaseID githubv4.Int
			URL        githubv4.URI

			Issues struct {
				Nodes []struct {
					ID              githubv4.ID
					Number          githubv4.Int
					Author          githubV4Actor
					PublishedAt     githubv4.DateTime
					LastEditedAt    *githubv4.DateTime
					Editor          *githubV4Actor
					Title           githubv4.String
					Body            githubv4.String
					ViewerCanUpdate githubv4.Boolean
				}
			} `graphql:"issues(first:$issueFirst)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
	}

	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String("golang"),
		"repositoryName":  githubv4.String("go"),
		"issueFirst":      githubv4.Int(100),
	}

	err := client.Query(context.Background(), &q, variables)
	if err != nil {
		return err
	}

	issues := q.Repository.Issues.Nodes
	idx, err := fuzzyfinder.FindMulti(
		issues,
		func(i int) string {
			return strconv.Itoa(int(issues[i].Number)) + " " + string(issues[i].Title)
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return fmt.Sprintf("Issue Number: %d (%s)\nTitle: %s\n\n%s",
				issues[i].Number,
				issues[i].ID,
				issues[i].Title,
				issues[i].Body,
			)
		}),
	)
	if err != nil {
		return err
	}

	fmt.Printf("selected: %v\n", idx)
	return nil
}
