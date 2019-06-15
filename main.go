package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

func main() {
	flag.Parse()

	err := run()
	if err != nil {
		log.Println(err)
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

			Issue struct {
				Author       githubV4Actor
				PublishedAt  githubv4.DateTime
				LastEditedAt *githubv4.DateTime
				Editor       *githubV4Actor
				Body         githubv4.String
				// ReactionGroups []struct {
				// 	Content githubv4.ReactionContent
				// 	Users   struct {
				// 		Nodes []struct {
				// 			Login githubv4.String
				// 		}
				//
				// 		TotalCount githubv4.Int
				// 	} `graphql:"users(first:10)"`
				// 	ViewerHasReacted githubv4.Boolean
				// }
				ViewerCanUpdate githubv4.Boolean

				// Comments struct {
				// 	Nodes []struct {
				// 		Body   githubv4.String
				// 		Author struct {
				// 			Login githubv4.String
				// 		}
				// 		Editor struct {
				// 			Login githubv4.String
				// 		}
				// 	}
				// 	PageInfo struct {
				// 		EndCursor   githubv4.String
				// 		HasNextPage githubv4.Boolean
				// 	}
				// } `graphql:"comments(first:$commentsFirst,after:$commentsAfter)"`
			} `graphql:"issue(number:$issueNumber)"`
		} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
		// Viewer struct {
		// 	Login      githubv4.String
		// 	CreatedAt  githubv4.DateTime
		// 	ID         githubv4.ID
		// 	DatabaseID githubv4.Int
		// }
		// RateLimit struct {
		// 	Cost      githubv4.Int
		// 	Limit     githubv4.Int
		// 	Remaining githubv4.Int
		// 	ResetAt   githubv4.DateTime
		// }
	}
	variables := map[string]interface{}{
		"repositoryOwner": githubv4.String("lighttiger2505"),
		"repositoryName":  githubv4.String("lab"),
		"issueNumber":     githubv4.Int(1),
		// "commentsFirst":   githubv4.NewInt(1),
		// "commentsAfter":   githubv4.NewString("Y3Vyc29yOjE5NTE4NDI1Ng=="),
	}
	err := client.Query(context.Background(), &q, variables)
	if err != nil {
		return err
	}
	printJSON(q)

	return nil
}

// printJSON prints v as JSON encoded with indent to stdout. It panics on any error.
func printJSON(v interface{}) {
	w := json.NewEncoder(os.Stdout)
	w.SetIndent("", "\t")
	err := w.Encode(v)
	if err != nil {
		panic(err)
	}
}
