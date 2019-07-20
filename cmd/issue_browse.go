package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/lighttiger2505/huc/internal/cmdutil"
	"github.com/lighttiger2505/huc/internal/config"
	"github.com/lighttiger2505/huc/internal/git"
	"github.com/lighttiger2505/huc/internal/github"
	"github.com/lighttiger2505/huc/internal/ui"
	"github.com/spf13/cobra"
)

var issueBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return browseIssueMain(cmd, args)
	},
}

func init() {
	issueCmd.AddCommand(issueBrowseCmd)
}

func browseIssueMain(cmd *cobra.Command, args []string) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return fmt.Errorf("cannot load config, %s", err)
	}
	remoteCollecter := git.NewRemoteCollecter(ui.NewBasicUi(), cfg, git.NewGitClient())

	pInfo, err := remoteCollecter.CollectTarget(
		"",
		"",
	)
	if err != nil {
		return err
	}

	number, err := getIssueNumber(args)
	if err != nil {
		return err
	}

	spProject := strings.Split(pInfo.Project, "/")
	issue, err := github.ShowIssue(pInfo.Token, spProject[0], spProject[1], number)
	if err != nil {
		return err
	}

	if err := browseIssue(pInfo, issue); err != nil {
		return err
	}
	return nil
}

func browseIssue(pInfo *git.GitLabProjectInfo, issue *github.Issue) error {
	b := &cmdutil.Browser{}
	selectedIssueNumber := int(issue.Number)
	url := strings.Join([]string{pInfo.SubpageUrl("issues"), strconv.Itoa(selectedIssueNumber)}, "/")

	if err := b.Open(url); err != nil {
		return err
	}
	return nil
}
