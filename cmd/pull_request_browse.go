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

var pullRequestBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return browsePullRequestMain(cmd, args)
	},
}

func init() {
	pullRequestCmd.AddCommand(pullRequestBrowseCmd)
}

func browsePullRequestMain(cmd *cobra.Command, args []string) error {
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

	number, err := getPullRequestNumber(args)
	if err != nil {
		return err
	}

	spProject := strings.Split(pInfo.Project, "/")
	pullRequest, err := github.ShowPullRequest(pInfo.Token, spProject[0], spProject[1], number)
	if err != nil {
		return err
	}

	if err := browsePullRequest(pInfo, pullRequest); err != nil {
		return err
	}
	return nil
}

func browsePullRequest(pInfo *git.GitLabProjectInfo, pullRequest *github.PullRequest) error {
	b := &cmdutil.Browser{}
	selectedPullRequestNumber := int(pullRequest.Number)
	url := strings.Join([]string{pInfo.SubpageUrl("pull"), strconv.Itoa(selectedPullRequestNumber)}, "/")

	if err := b.Open(url); err != nil {
		return err
	}
	return nil
}
