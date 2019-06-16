package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return run()
	},
}

func init() {
	rootCmd.AddCommand(issueCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// issueCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// issueCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

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

func run() error {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_GRAPHQL_TEST_TOKEN")},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	client := githubv4.NewClient(httpClient)

	var q struct {
		Repository struct {
			DatabaseID githubv4.Int
			URL        githubv4.URI

			Issues struct {
				Nodes []Issue
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
			return issues[i].ToString()
		}),
	)
	if err != nil {
		return err
	}

	b := &Browser{}
	url := fmt.Sprintf("https://github.com/golang/go/issues/%d", idx[0])

	if err := b.Open(url); err != nil {
		return err
	}

	fmt.Printf("Open selected issue: %v\n", idx[0])

	return nil
}

type URLOpener interface {
	Open(url string) error
}

type Browser struct{}

func (b *Browser) Open(url string) error {
	browser := searchBrowserLauncher(runtime.GOOS)
	c := exec.Command(browser, url)
	if err := c.Run(); err != nil {
		return err
	}
	return nil
}

func searchBrowserLauncher(goos string) (browser string) {
	switch goos {
	case "darwin":
		browser = "open"
	case "windows":
		browser = "cmd /c start"
	default:
		candidates := []string{
			"xdg-open",
			"cygstart",
			"x-www-browser",
			"firefox",
			"opera",
			"mozilla",
			"netscape",
		}
		for _, b := range candidates {
			path, err := exec.LookPath(b)
			if err == nil {
				browser = path
				break
			}
		}
	}
	return browser
}
