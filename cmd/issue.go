package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/lighttiger2505/huc/internal/cmdutil"
	"github.com/lighttiger2505/huc/internal/config"
	"github.com/lighttiger2505/huc/internal/git"
	"github.com/lighttiger2505/huc/internal/github"
	"github.com/lighttiger2505/huc/internal/ui"
	"github.com/shurcooL/githubv4"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
		return findIssue(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.Flags().IntP("num", "n", 50, "Number of lists to display.")
	issueCmd.Flags().StringP("direction", "", "DESC", "To sort order. Can be either ASC or DESC")
	issueCmd.Flags().StringP("sort", "", "CREATED_AT", "What to sort results by. Can be either COMMENTS, CREATED_AT or UPDATED_AT")
	issueCmd.Flags().StringP("states", "", "OPEN", "Indicates the state of the issues to display. OPEN or CLOSED")
	issueCmd.Flags().StringP("labels", "", "", "A list of comma separated label names.")
}

func findIssue(cmd *cobra.Command, args []string) error {
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

	opt, err := toListProjectIssueOption(cmd.Flags())
	if err != nil {
		return err
	}

	spProject := strings.Split(pInfo.Project, "/")
	issues, err := github.ListIssue(pInfo.Token, spProject[0], spProject[1], opt)
	if err != nil {
		return err
	}

	idx, err := fuzzyfinder.Find(
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
		if err.Error() == fuzzyfinder.ErrAbort.Error() {
			return nil
		}
		return err
	}

	b := &cmdutil.Browser{}
	selectedIssueNumber := int(issues[int(idx)].Number)
	url := strings.Join([]string{pInfo.SubpageUrl("issues"), strconv.Itoa(selectedIssueNumber)}, "/")

	if err := b.Open(url); err != nil {
		return err
	}

	return nil
}

func toListProjectIssueOption(flags *pflag.FlagSet) (*github.ListProjectIssueOption, error) {
	num, err := flags.GetInt("num")
	if err != nil {
		return nil, err
	}

	direction, err := flags.GetString("direction")
	if err != nil {
		return nil, err
	}
	var directionOpt githubv4.OrderDirection
	switch direction {
	case "ASC":
		directionOpt = githubv4.OrderDirectionAsc
	case "DESC":
		directionOpt = githubv4.OrderDirectionDesc
	default:
		return nil, fmt.Errorf("Invalid issue order option, %s", direction)
	}

	sort, err := flags.GetString("sort")
	if err != nil {
		return nil, err
	}
	var sortOpt githubv4.IssueOrderField
	switch sort {
	case "COMMENTS":
		sortOpt = githubv4.IssueOrderFieldComments
	case "CREATED_AT":
		sortOpt = githubv4.IssueOrderFieldCreatedAt
	case "UPDATED_AT":
		sortOpt = githubv4.IssueOrderFieldUpdatedAt
	default:
		return nil, fmt.Errorf("Invalid issue sort option, %s", sort)
	}

	states, err := flags.GetString("states")
	if err != nil {
		return nil, err
	}
	var statesOpt githubv4.IssueState
	switch states {
	case "OPEN":
		statesOpt = githubv4.IssueStateOpen
	case "CLOSED":
		statesOpt = githubv4.IssueStateClosed
	default:
		return nil, fmt.Errorf("Invalid issue sort option, %s", states)
	}

	return &github.ListProjectIssueOption{
		Num:       num,
		Sort:      sortOpt,
		Direction: directionOpt,
		States:    statesOpt,
	}, nil
}
