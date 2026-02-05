package query

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/flanksource/duty/cache"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/models"
	"github.com/flanksource/gomplate/v3/conv"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

type Kustomize struct {
	Path string `json:"path"`
	File string `json:"file"`
}

func (t *Kustomize) AsMap() map[string]any {
	return map[string]any{
		"path": t.Path,
		"file": t.File,
	}
}

type Git struct {
	Link   string `json:"link"`
	File   string `json:"file"`
	Dir    string `json:"dir"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

func (t *Git) AsMap() map[string]any {
	return map[string]any{
		"file":   t.File,
		"link":   t.Link,
		"dir":    t.Dir,
		"url":    t.URL,
		"branch": t.Branch,
	}
}

type GitOpsSource struct {
	Git       Git       `json:"git"`
	Kustomize Kustomize `json:"kustomize"`
}

func (t *GitOpsSource) AsMap() map[string]any {
	m := t.Git.AsMap()
	// FIXME - git duplicated for backwards compatibility remove once all playbooks updated
	m["git"] = t.Git.AsMap()
	m["kustomize"] = t.Kustomize.AsMap()
	return m
}

func getOrigin(ci *models.ConfigItem) (map[string]any, error) {
	origin := make(map[string]any)
	_origin := ci.NestedString("metadata", "annotations", "config.kubernetes.io/origin")
	if _origin != "" {
		if err := yaml.Unmarshal([]byte(_origin), &origin); err != nil {
			return origin, err
		}
	}
	return origin, nil
}

// GitOpsSource rarely changes, so we cache it for 24 hours
var gitOpsSourceCache = cache.NewCache[GitOpsSource]("GetGitOpsSource", time.Hour*24)

func GetGitOpsSource(ctx context.Context, id uuid.UUID) (GitOpsSource, error) {
	if val, err := gitOpsSourceCache.Get(ctx, id); err == nil {
		return val, nil
	}

	var source GitOpsSource
	if id == uuid.Nil {
		return source, nil
	}

	ci, err := GetCachedConfig(ctx, id.String())
	if err != nil {
		return source, err
	}
	if ci == nil {
		return source, nil
	}

	gitRepoRelationType := "Kubernetes::Kustomization/Kubernetes::GitRepository"
	if lo.FromPtr(ci.Type) == "Kubernetes::Kustomization" {
		gitRepoRelationType = "Kubernetes::GitRepository"
	}

	gitRepos := TraverseConfig(ctx, id.String(), gitRepoRelationType, string(models.RelatedConfigTypeIncoming))
	if gitRepo := lo.FirstOrEmpty(gitRepos); gitRepo.Config != nil {
		source.Git.URL = gitRepo.NestedString("spec", "url")
		// These are in order of precedence for fluxcd.io/GitRepository
		source.Git.Branch = lo.CoalesceOrEmpty(
			gitRepo.NestedString("spec", "ref", "commit"),
			gitRepo.NestedString("spec", "ref", "name"),
			gitRepo.NestedString("spec", "ref", "semver"),
			gitRepo.NestedString("spec", "ref", "tag"),
			gitRepo.NestedString("spec", "ref", "branch"),
		)
	}

	if lo.FromPtr(ci.Type) == "Kubernetes::Kustomization" {
		source.Kustomize.Path = ci.NestedString("spec", "path")
		source.Kustomize.File = filepath.Join(source.Kustomize.Path, "kustomization.yaml")
	} else {
		kustomization := TraverseConfig(ctx, id.String(), "Kubernetes::Kustomization", string(models.RelatedConfigTypeIncoming))
		if len(kustomization) > 0 && kustomization[0].Config != nil {
			source.Kustomize.Path = kustomization[0].NestedString("spec", "path")
			source.Kustomize.File = filepath.Join(source.Kustomize.Path, "kustomization.yaml")
		}
	}

	origin, _ := getOrigin(ci)
	if path, ok := origin["path"]; ok {
		source.Git.File = filepath.Join(source.Kustomize.Path, path.(string))
		source.Git.Dir = filepath.Dir(source.Git.File)
	}

	if strings.Contains(source.Git.URL, "github.com") {
		source.Git.Link = fmt.Sprintf("https://%s/tree/%s/%s", stripScheme(source.Git.URL), source.Git.Branch, source.Git.File)
	}

	_ = gitOpsSourceCache.Set(ctx, id, source)
	return source, nil
}

func stripScheme(uri string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	u.Scheme = ""
	u.User = nil
	return strings.TrimPrefix(strings.ReplaceAll(u.String(), ".git", ""), "//")
}

func gitopsSourceCELFunction() func(ctx context.Context) cel.EnvOption {
	return func(ctx context.Context) cel.EnvOption {
		return cel.Function("gitops.source",
			cel.Overload("gitops.source_interface{}",
				[]*cel.Type{cel.DynType},
				cel.DynType,
				cel.UnaryBinding(func(arg ref.Val) ref.Val {
					id, err := getConfigId(arg.Value())
					if err != nil {
						ctx.Errorf("could not find id: %v", err)
						return types.DefaultTypeAdapter.NativeToValue((&GitOpsSource{}).AsMap())
					}

					source, err := GetGitOpsSource(ctx, id)
					if err != nil {
						return types.WrapErr(err)
					}

					return types.DefaultTypeAdapter.NativeToValue(source.AsMap())
				}),
			),
		)
	}
}

func getConfigId(id any) (uuid.UUID, error) {
	switch v := id.(type) {
	case string:
		return uuid.Parse(v)
	case uuid.UUID:
		return v, nil
	case models.ConfigItem:
		return v.ID, nil
	case map[string]string:
		if v, ok := v["id"]; ok {
			return uuid.Parse(v)
		}
	case map[string]any:
		if v, ok := v["id"]; ok {
			switch v2 := v.(type) {
			case uuid.UUID:
				return v2, nil
			case []byte:
				return uuid.UUID(v2), nil
			case string:
				return uuid.Parse(v2)
			default:
				return uuid.Parse(conv.ToString(v2))
			}
		}
	}
	return uuid.Nil, fmt.Errorf("unknown uuid type: %t", id)
}

func gitopsSourceTemplateFunction() func(ctx context.Context) any {
	return func(ctx context.Context) any {
		return func(args ...any) map[string]any {
			var source GitOpsSource
			if len(args) < 1 {
				return source.AsMap()
			}

			id, err := getConfigId(args[0])
			if err != nil {
				ctx.Errorf("could not find id '%s' from %v: %v", id, args[0], err)
				return source.AsMap()
			}

			source, err = GetGitOpsSource(ctx, id)
			if err != nil {
				ctx.Errorf("%s", err)
			}
			return source.AsMap()
		}
	}
}

func init() {
	context.CelEnvFuncs["gitops.source"] = gitopsSourceCELFunction()
	context.TemplateFuncs["gitops_source"] = gitopsSourceTemplateFunction()
}
