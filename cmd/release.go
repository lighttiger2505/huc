package cmd

import (
	"fmt"
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

var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return findRelease(cmd, args)
	},
	Aliases: []string{"r"},
}

func init() {
	rootCmd.AddCommand(releaseCmd)
	releaseCmd.Flags().IntP("num", "n", 50, "Number of lists to display.")
	releaseCmd.Flags().StringP("direction", "", "DESC", "To sort order. Can be either ASC or DESC")
	releaseCmd.Flags().StringP("sort", "", "CREATED_AT", "What to sort results by. Can be either COMMENTS, CREATED_AT or UPDATED_AT")
}

func findRelease(cmd *cobra.Command, args []string) error {
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

	opt, err := toListProjectReleasesOption(cmd.Flags())
	if err != nil {
		return err
	}

	spProject := strings.Split(pInfo.Project, "/")
	releases, err := github.ListRelease(pInfo.Token, spProject[0], spProject[1], opt)
	if err != nil {
		return err
	}

	idx, err := fuzzyfinder.Find(
		releases,
		func(i int) string {
			return string(releases[i].Name)
		},
		fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
			if i == -1 {
				return ""
			}
			return releases[i].ToString()
		}),
	)

	if err != nil {
		if err.Error() == fuzzyfinder.ErrAbort.Error() {
			return nil
		}
		return err
	}

	b := &cmdutil.Browser{}
	selectedIssueNumber := string(releases[int(idx)].TagName)
	url := strings.Join([]string{pInfo.SubpageUrl("releases/tag"), selectedIssueNumber}, "/")

	if err := b.Open(url); err != nil {
		return err
	}

	return nil
}

func toListProjectReleasesOption(flags *pflag.FlagSet) (*github.ListProjectReleaseOption, error) {
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
		return nil, fmt.Errorf("Invalid release order option, %s", direction)
	}

	sort, err := flags.GetString("sort")
	if err != nil {
		return nil, err
	}
	var sortOpt githubv4.ReleaseOrderField
	switch sort {
	case "NAME":
		sortOpt = githubv4.ReleaseOrderFieldName
	case "CREATED_AT":
		sortOpt = githubv4.ReleaseOrderFieldCreatedAt
	default:
		return nil, fmt.Errorf("Invalid release sort option, %s", sort)
	}

	return &github.ListProjectReleaseOption{
		Num:       num,
		Sort:      sortOpt,
		Direction: directionOpt,
	}, nil
}
