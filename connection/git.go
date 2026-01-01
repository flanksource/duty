package connection

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/commons/utils"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/samber/lo"
	ssh2 "golang.org/x/crypto/ssh"

	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/types"
)

const (
	ServiceGithub = "github"
	ServiceGitlab = "gitlab"
)

type GitClient struct {
	Auth                transport.AuthMethod
	URL                 string
	Owner, Repo, Branch string
	Depth               int
	AzureDevops         bool
}

func (gitClient GitClient) GetContext() map[string]any {
	if uri, err := url.Parse(gitClient.URL); err == nil {
		return map[string]any{
			"url":    uri.Redacted(),
			"branch": gitClient.Branch,
		}
	}
	return map[string]any{
		"url":    "redacted",
		"owner":  gitClient.Owner,
		"repo":   gitClient.Repo,
		"branch": gitClient.Branch,
	}
}

func (gitClient GitClient) GetShortURL() string {
	u, err := url.Parse(gitClient.URL)
	if err != nil {
		return ""
	}
	u.Scheme = ""
	u.RawQuery = ""
	return strings.TrimLeft(u.Redacted(), "/")
}

func (gitClient GitClient) LoggerName() string {
	if gitClient.Branch != "" && gitClient.Branch != "main" {
		return fmt.Sprintf("%s@%s", gitClient.GetShortURL(), gitClient.Branch)
	}
	return gitClient.GetShortURL()
}

func (gitClient *GitClient) Clone(ctx context.Context, dir string) (map[string]any, error) {

	if gitClient.AzureDevops {
		transport.UnsupportedCapabilities = []capability.Capability{
			capability.ThinPack,
		}
	}

	ctx = ctx.WithObject(*gitClient)
	if ctx.Logger.IsLevelEnabled(4) {
		ctx.Logger.V(4).Infof("cloning to %s", dir)
	} else {
		ctx.Tracef("cloning")
	}
	extra := map[string]any{
		"git": gitClient.GetShortURL(),
	}

	repo, err := git.PlainCloneContext(ctx, dir, false, &git.CloneOptions{
		URL:           gitClient.URL,
		Progress:      ctx.Logger.V(4).WithFilter("Compressing objects", "Counting objects"),
		Auth:          gitClient.Auth,
		ReferenceName: plumbing.NewBranchReferenceName(gitClient.Branch),
		Depth:         gitClient.Depth,
	})

	if errors.Is(err, git.ErrRepositoryAlreadyExists) {
		repo, err = git.PlainOpen(dir)
		if err != nil {
			return extra, ctx.Oops().Wrapf(err, "unable to open repository")
		}

		tree, err := repo.Worktree()
		if err != nil {
			return extra, ctx.Oops().Wrapf(err, "unable to open worktree")
		}

		ctx.Logger.V(4).Infof("fetching ")
		if err := repo.FetchContext(ctx, &git.FetchOptions{
			Progress:  ctx.Logger.V(4).WithFilter("Compressing objects", "Counting objects"),
			RemoteURL: gitClient.URL,
			Force:     true,
			Prune:     true,
			Auth:      gitClient.Auth,
			Depth:     gitClient.Depth}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return extra, ctx.Oops().Wrapf(err, "error during git fetch")
		}

		refName := plumbing.NewRemoteReferenceName("origin", gitClient.Branch)
		if remote, err := repo.Remote("origin"); err == nil {
			list, err := remote.List(&git.ListOptions{
				Auth: gitClient.Auth,
			})
			if err != nil {
				return extra, ctx.Oops().Wrapf(err, "error during git remote ls")
			}

			for _, ref := range list {
				if ref.Name().Short() == gitClient.Branch {
					refName = ref.Name()
					ctx.Logger.V(4).Infof("found ref %s matching %s", refName, gitClient.Branch)
				}
			}

		}

		if err := tree.Checkout(&git.CheckoutOptions{
			Branch: refName,
			Force:  true,
		}); err != nil {
			return extra, ctx.Oops().Wrapf(err, "error during git checkout")
		}
	} else if err != nil {
		return extra, ctx.Oops().Wrapf(err, "error during git clone")
	}

	if commit, err := repo.Head(); err != nil {
		return extra, ctx.Oops().Wrapf(err, "unable to get HEAD commit")
	} else {
		extra["commit"] = commit.Hash().String()

		if iter, err := repo.Log(&git.LogOptions{From: commit.Hash()}); err == nil {
			if commit, err := iter.Next(); err != nil {
				return extra, ctx.Oops().Wrapf(err, "unable to get HEAD commit")
			} else {
				ctx.Logger.V(4).Infof("checked out commit: %s (%s)", strings.Split(commit.Message, "\n")[0], commit.Hash.String()[0:8])
			}
		}
	}

	return extra, nil
}

// +kubebuilder:object:generate=true
type GitConnection struct {
	URL         string        `yaml:"url,omitempty" json:"url,omitempty"`
	Connection  string        `yaml:"connection,omitempty" json:"connection,omitempty"`
	Username    *types.EnvVar `yaml:"username,omitempty" json:"username,omitempty"`
	Password    *types.EnvVar `yaml:"password,omitempty" json:"password,omitempty"`
	Certificate *types.EnvVar `yaml:"certificate,omitempty" json:"certificate,omitempty"`
	// Type of connection e.g. github, gitlab
	Type   string `yaml:"type,omitempty" json:"type,omitempty"`
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`

	Depth *int `json:"depth,omitempty"`
	// Destination is the full path to where the contents of the URL should be downloaded to.
	// If left empty, the sha256 hash of the URL will be used as the dir name.
	//
	// Deprecated: no similar functionality available. This depends on the use case
	Destination *string `yaml:"destination,omitempty" json:"destination,omitempty"`
}

func (git GitConnection) GetURL() types.EnvVar {
	return types.EnvVar{ValueStatic: git.URL}
}

func (git GitConnection) GetUsername() types.EnvVar {
	return utils.Deref(git.Username)
}

func (git GitConnection) GetPassword() types.EnvVar {
	return utils.Deref(git.Password)
}

func (git GitConnection) GetCertificate() types.EnvVar {
	return utils.Deref(git.Certificate)
}

func (c *GitConnection) HydrateConnection(ctx context.Context) error {
	ctx.Logger.V(9).Infof("Hydrating GitConnection %s", logger.Pretty(*c))

	if c.Connection != "" {
		conn, err := ctx.HydrateConnectionByURL(c.Connection)
		if err != nil {
			return err
		}
		if conn != nil {
			if conn.Username != "" {
				c.Username = &types.EnvVar{ValueStatic: conn.Username}
			}
			if conn.Password != "" {
				c.Password = &types.EnvVar{ValueStatic: conn.Password}
			}
			if conn.Certificate != "" {
				c.Certificate = &types.EnvVar{ValueStatic: conn.Certificate}
			}
			if c.URL == "" {
				c.URL = conn.URL
			}
		}
	}

	if uri, err := url.Parse(c.URL); err == nil {
		if uri.Scheme == "" {
			uri.Scheme = "https"
			c.URL = uri.String()
		}
	}

	if c.Username == nil {
		c.Username = &types.EnvVar{}
	}

	if c.Password == nil {
		c.Password = &types.EnvVar{}
	}

	if c.Certificate == nil {
		c.Certificate = &types.EnvVar{}
	}

	if username, err := ctx.GetEnvValueFromCache(*c.Username, ctx.GetNamespace()); err != nil {
		return fmt.Errorf("could not parse username: %v", err)
	} else if username != "" {
		c.Username.ValueStatic = username
	}

	if password, err := ctx.GetEnvValueFromCache(*c.Password, ctx.GetNamespace()); err != nil {
		return fmt.Errorf("could not parse password: %w", err)
	} else if password != "" {
		c.Password.ValueStatic = password
	}

	if certificate, err := ctx.GetEnvValueFromCache(*c.Certificate, ctx.GetNamespace()); err != nil {
		return fmt.Errorf("could not parse certificate: %v", err)
	} else if certificate != "" {
		c.Certificate.ValueStatic = certificate
	}

	ctx.Logger.V(9).Infof("Hydrated GitConnection %s", logger.Pretty(*c))

	return nil
}

func CreateGitConfig(ctx context.Context, conn *GitConnection) (*GitClient, error) {
	config := &GitClient{
		URL:    conn.URL,
		Depth:  lo.Ternary(conn.Depth == nil, 1, lo.FromPtr(conn.Depth)),
		Branch: lo.CoalesceOrEmpty(conn.Branch, "main"),
	}

	if uri, err := url.Parse(conn.URL); err == nil {
		if ref := uri.Query().Get("ref"); ref != "" {
			config.Branch = ref
		}
		if depth := uri.Query().Get("depth"); depth != "" {
			depthInt, err := strconv.Atoi(depth)
			if err != nil {
				return nil, err
			}
			config.Depth = depthInt
		}
		// strip of any query parameters
		uri.RawQuery = ""
		config.URL = uri.String()
	}

	if owner, repo, ok := parseGenericRepoURL(conn.URL, "github.com", false); ok {
		config.Owner = owner
		config.Repo = repo
	} else if owner, repo, ok := parseGenericRepoURL(conn.URL, "gitlab.com", conn.Type == ServiceGitlab); ok {
		config.Owner = owner
		config.Repo = repo
	} else if azureOrg, azureProject, repo, ok := parseAzureDevopsRepo(conn.URL); ok {
		config.Owner = fmt.Sprintf("%s/%s", azureOrg, azureProject)
		config.Repo = repo
		config.AzureDevops = true
	}
	if strings.HasPrefix(conn.URL, "ssh://") {
		sshURL := conn.URL[6:]
		user := strings.Split(sshURL, "@")[0]

		publicKeys, err := ssh.NewPublicKeys(user, []byte(conn.Certificate.ValueStatic), conn.Password.ValueStatic)
		if err != nil {
			return nil, ctx.Oops().Wrapf(err, "failed to create public keys")
		}
		publicKeys.HostKeyCallback = ssh2.InsecureIgnoreHostKey()
		config.Auth = publicKeys

	} else {
		config.Auth = &http.BasicAuth{
			Username: conn.Username.ValueStatic,
			Password: conn.Password.ValueStatic,
		}
	}

	return config, nil
}

var azureDevopsRepoURLRegexp = regexp.MustCompile(`^https:\/\/[a-zA-Z0-9_-]+@dev\.azure\.com\/([a-zA-Z0-9_-]+)\/([a-zA-Z0-9_-]+)\/_git\/([a-zA-Z0-9_-]+)`)

func parseAzureDevopsRepo(url string) (org, project, repo string, ok bool) {
	matches := azureDevopsRepoURLRegexp.FindStringSubmatch(url)
	if len(matches) != 4 {
		return "", "", "", false
	}

	return matches[1], matches[2], matches[3], true
}

// parseGenericRepoURL parses a URL into owner and repo.
//   - custom: true if the repo has custom domain
func parseGenericRepoURL(repoURL, host string, custom bool) (owner string, repo string, ok bool) {
	parsed, err := url.Parse(repoURL)
	if err != nil {
		return "", "", false
	}

	if !custom && parsed.Hostname() != host {
		return "", "", false
	}

	path := strings.TrimSuffix(parsed.Path, ".git")
	path = strings.TrimPrefix(path, "/")
	paths := strings.Split(path, "/")
	if len(paths) != 2 {
		return "", "", false
	}

	return paths[0], paths[1], true
}
