package git

import (
	"fmt"
	"strings"

	"github.com/lighttiger2505/huc/internal/config"
	"github.com/lighttiger2505/huc/internal/ui"
)

type Collecter interface {
	CollectTarget(project, profile string) (*GitLabProjectInfo, error)
}

type RemoteCollecter struct {
	UI        ui.UI
	GitClient Client
	Cfg       *config.Config
}

type GitLabProjectInfo struct {
	Domain        string
	Project       string
	Token         string
	CurrentBranch string
	Profile       *config.Profile
}

func (r *GitLabProjectInfo) BaseUrl() string {
	return "https://" + r.Domain
}

func (r *GitLabProjectInfo) ApiUrl() string {
	return strings.Join([]string{r.BaseUrl(), "api", "v4"}, "/")
}

func (r *GitLabProjectInfo) SubpageUrl(subpage string) string {
	return strings.Join([]string{r.RepositoryUrl(), subpage}, "/")
}

func (r *GitLabProjectInfo) RepositoryUrl() string {
	return strings.Join([]string{r.BaseUrl(), r.Project}, "/")
}

func (r *GitLabProjectInfo) BranchUrl(branch string) string {
	return strings.Join([]string{r.RepositoryUrl(), "tree", branch}, "/")
}

func (r *GitLabProjectInfo) BranchPath(branch string, path string) string {
	return strings.Join([]string{r.BranchUrl(branch), path}, "/")
}

func (r *GitLabProjectInfo) BranchFileWithLine(branch string, path string, line string) string {
	return strings.Join([]string{r.BranchPath(branch, path), line}, "/")
}

func (r *GitLabProjectInfo) Subpage(subpage string) string {
	return strings.Join([]string{r.RepositoryUrl(), subpage}, "/")
}

func NewRemoteCollecter(ui ui.UI, cfg *config.Config, gitClient Client) Collecter {
	return &RemoteCollecter{
		UI:        ui,
		Cfg:       cfg,
		GitClient: gitClient,
	}
}

func (c *RemoteCollecter) CollectTarget(project, profile string) (*GitLabProjectInfo, error) {
	pInfo := &GitLabProjectInfo{}
	var err error

	isGitDir, err := IsGitDirReverseTop()
	if err != nil {
		return nil, err
	}
	if isGitDir {
		pInfo = c.collectTargetByDefaultConfig(pInfo)
		pInfo, err = c.collectTargetByLocalRepository(pInfo)
		if err != nil {
			return nil, err
		}
		pInfo, err = c.collectTargetByArgs(pInfo, project, profile)
		if err != nil {
			return nil, err
		}
	} else {
		pInfo = c.collectTargetByDefaultConfig(pInfo)
		pInfo, err = c.collectTargetByArgs(pInfo, project, profile)
		if err != nil {
			return nil, err
		}
	}

	return pInfo, nil
}

func (c *RemoteCollecter) collectTargetByDefaultConfig(pInfo *GitLabProjectInfo) *GitLabProjectInfo {
	if c.Cfg.DefalutProfile == "" {
		return pInfo
	}
	profile := c.Cfg.GetDefaultProfile()
	pInfo.Profile = profile
	pInfo.Domain = c.Cfg.DefalutProfile
	pInfo.Token = profile.Token

	if profile.DefaultProject == "" {
		return pInfo
	}
	pInfo.Project = profile.DefaultProject

	return pInfo
}

func (c *RemoteCollecter) collectTargetByLocalRepository(pInfo *GitLabProjectInfo) (*GitLabProjectInfo, error) {
	gitRemotes, err := c.GitClient.RemoteInfos()
	if err != nil {
		return nil, err
	}

	gitlabRemotes := filterHasGitlabDomain(gitRemotes, c.Cfg)
	if len(gitlabRemotes) == 0 {
		return nil, fmt.Errorf("Not found gitlab remote repository")
	}
	gitlabRemotes = excludeDuplicateDomain(gitlabRemotes)
	targetRepo := gitlabRemotes[0]

	var domain, token string

	domain = targetRepo.Domain
	if !c.Cfg.HasDomain(domain) {
		c.UI.Message(fmt.Sprintf("Not found this domain [%s].", domain))
		c.Cfg.SetProfile(domain, config.Profile{})
		if err := c.Cfg.Save(); err != nil {
			return nil, err
		}
		c.UI.Message("Saved profile.")
	}

	token = c.Cfg.GetToken(domain)
	if token == "" {
		c.UI.Message(fmt.Sprintf("Not found private token in the domain [%s].", domain))
		token, err = c.UI.Ask("Please enter GitLab private token:")
		if err != nil {
			return nil, fmt.Errorf("cannot read private token, %s", err)
		}

		c.Cfg.SetToken(domain, token)
		if err := c.Cfg.Save(); err != nil {
			return nil, err
		}
		c.UI.Message("Saved private Token.")
	}

	profile, err := c.Cfg.GetProfile(domain)
	if err != nil {
		return nil, err
	}

	pInfo.Profile = profile
	pInfo.Domain = domain
	pInfo.Token = token
	pInfo.Project = targetRepo.RepositoryFullName()

	currentBranch, err := c.GitClient.CurrentRemoteBranch()
	if err != nil {
		return nil, err
	}
	pInfo.CurrentBranch = currentBranch

	return pInfo, nil
}

func (c *RemoteCollecter) collectTargetByArgs(pInfo *GitLabProjectInfo, project, profile string) (*GitLabProjectInfo, error) {
	if profile != "" {
		p, err := c.Cfg.GetProfile(profile)
		if err != nil {
			return nil, err
		}
		pInfo.Profile = p
		pInfo.Domain = profile
		pInfo.Token = p.Token
	}

	if project != "" {
		pInfo.Project = project
	}

	return pInfo, nil
}

func filterHasGitlabDomain(remoteInfos []*RemoteInfo, cfg *config.Config) []*RemoteInfo {
	gitlabRemotes := []*RemoteInfo{}
	for _, remoteInfo := range remoteInfos {
		if strings.HasPrefix(remoteInfo.Domain, "github") {
			gitlabRemotes = append(gitlabRemotes, remoteInfo)
		} else if cfg.HasDomain(remoteInfo.Domain) {
			gitlabRemotes = append(gitlabRemotes, remoteInfo)
		}
	}
	return gitlabRemotes
}

func excludeDuplicateDomain(remotes []*RemoteInfo) []*RemoteInfo {
	domainRemotesMap := map[string][]*RemoteInfo{}
	for _, remote := range remotes {
		domain := remote.Domain
		domainRemotesMap[domain] = append(domainRemotesMap[domain], remote)
	}

	processedRemotes := []*RemoteInfo{}
	for _, v := range domainRemotesMap {
		var tmpRemote = v[0]
		for _, remote := range v {
			if remote.Remote == "origin" {
				tmpRemote = remote
				break
			}
		}
		processedRemotes = append(processedRemotes, tmpRemote)
	}
	return processedRemotes
}

type MockCollecter struct{}

func (m *MockCollecter) CollectTarget(project, profile string) (*GitLabProjectInfo, error) {
	return &GitLabProjectInfo{
		Domain:  "domain",
		Project: "project",
		Token:   "token",
		Profile: &config.Profile{},
	}, nil
}
