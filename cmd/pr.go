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

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return findPullRequest(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(prCmd)
	prCmd.Flags().IntP("num", "n", 50, "Number of lists to display.")
	prCmd.Flags().StringP("direction", "", "DESC", "To sort order. Can be either ASC or DESC")
	prCmd.Flags().StringP("sort", "", "CREATED_AT", "What to sort results by. Can be either COMMENTS, CREATED_AT or UPDATED_AT")
	prCmd.Flags().StringP("states", "", "OPEN", "Indicates the state of the pull requests to display. OPEN or CLOSED, MERGED")
	prCmd.Flags().StringP("labels", "", "", "A list of comma separated label names.")
}

func findPullRequest(cmd *cobra.Command, args []string) error {
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

	opt, err := toListProjectPullReqeustOption(cmd.Flags())
	if err != nil {
		return err
	}

	spProject := strings.Split(pInfo.Project, "/")
	pullRequests, err := github.ListPullRequest(pInfo.Token, spProject[0], spProject[1], opt)
	if err != nil {
		return err
	}

	idx, err := fuzzyfinder.Find(
		pullRequests,
		func(i int) string {
			return strconv.Itoa(int(pullRequests[i].Number)) + " " + string(pullRequests[i].Title)
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return pullRequests[i].ToString()
		}),
	)

	if err != nil {
		if err.Error() == fuzzyfinder.ErrAbort.Error() {
			return nil
		}
		return err
	}

	b := &cmdutil.Browser{}
	selectedIssueNumber := int(pullRequests[int(idx)].Number)
	url := strings.Join([]string{pInfo.SubpageUrl("pull"), strconv.Itoa(selectedIssueNumber)}, "/")

	if err := b.Open(url); err != nil {
		return err
	}

	return nil
}

func toListProjectPullReqeustOption(flags *pflag.FlagSet) (*github.ListProjectPullRequestOption, error) {
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
	var statesOpt githubv4.PullRequestState
	switch states {
	case "OPEN":
		statesOpt = githubv4.PullRequestStateOpen
	case "MERGED":
		statesOpt = githubv4.PullRequestStateMerged
	case "CLOSED":
		statesOpt = githubv4.PullRequestStateClosed
	default:
		return nil, fmt.Errorf("Invalid issue sort option, %s", states)
	}

	return &github.ListProjectPullRequestOption{
		Num:       num,
		Sort:      sortOpt,
		Direction: directionOpt,
		States:    statesOpt,
	}, nil
}
