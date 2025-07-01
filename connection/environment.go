package connection

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	osExec "os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"
	"github.com/flanksource/duty/kubernetes"
	"github.com/flanksource/duty/models"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	textTemplate "text/template"
)

// kubeEnvVars holds a list of environment variables that are commonly used
// to configure access to the default Kubernetes cluster
var kubeEnvVars = []string{
	"KUBECONFIG",
	"KUBERNETES_SERVICE_HOST",
	"KUBERNETES_SERVICE_PORT",
	"KUBERNETES_PORT_443_TCP",
	"KUBERNETES_SERVICE_PORT_HTTPS",
	"KUBERNETES_PORT_443_TCP_PROTO",
	"KUBERNETES_PORT_443_TCP_ADDR",
	"KUBERNETES_PORT",
	"KUBERNETES_PORT_443_TCP_PORT",
}

// +kubebuilder:object:generate=true
type ExecConnections struct {
	FromConfigItem *string `yaml:"fromConfigItem,omitempty" json:"fromConfigItem,omitempty" template:"true"`

	// EKSPodIdentity when enabled will allow access to AWS_* env vars
	EKSPodIdentity bool `json:"eksPodIdentity,omitempty"`

	// ServiceAccount when enabled will allow access to KUBERNETES env vars
	ServiceAccount bool `json:"serviceAccount,omitempty"`

	Kubernetes *KubernetesConnection `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty"`
	AWS        *AWSConnection        `yaml:"aws,omitempty" json:"aws,omitempty"`
	GCP        *GCPConnection        `yaml:"gcp,omitempty" json:"gcp,omitempty"`
	Azure      *AzureConnection      `yaml:"azure,omitempty" json:"azure,omitempty"`
}

func saveConfig(cwd string, configTemplate *textTemplate.Template, view any) (string, error) {
	dirPath := filepath.Join(cwd, ".creds", fmt.Sprintf("cred-%d", rand.Intn(10000000)))
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", err
	}

	configPath := fmt.Sprintf("%s/credentials", dirPath)
	logger.Tracef("Creating credentials file: %s", configPath)

	file, err := os.Create(configPath)
	if err != nil {
		return configPath, err
	}
	defer file.Close()

	if err := configTemplate.Execute(file, view); err != nil {
		return configPath, err
	}

	return configPath, nil
}

var (
	awsConfigTemplate        *textTemplate.Template
	kubernetesConfigTemplate *textTemplate.Template
	gcloudConfigTemplate     *textTemplate.Template
)

func init() {
	awsConfigTemplate = textTemplate.Must(textTemplate.New("").Parse(`[default]
aws_access_key_id = {{.AccessKey.ValueStatic}}
aws_secret_access_key = {{.SecretKey.ValueStatic}}
{{if .SessionToken.ValueStatic}}aws_session_token={{.SessionToken.ValueStatic}}{{end}}
`))

	gcloudConfigTemplate = textTemplate.Must(textTemplate.New("").Parse(`{{.Credentials}}`))

	kubernetesConfigTemplate = textTemplate.Must(textTemplate.New("").Parse(`{{.Kubeconfig.ValueStatic}}`))
}

type ConnectionSetupResult struct {
	Sources   []string `json:"source,omitempty"`
	EnvVars   []string `json:"envVars,omitempty"`
	ApiServer string   `json:"kubeApiServer,omitempty"`

	Cleanup func() error `json:"-"`
}

func injectEksPodIdentity(ctx context.Context, cmd *osExec.Cmd) {
	ctx.Logger.V(3).Infof("Injecting EKS Pod Identity")

	for _, env := range os.Environ() {
		key, _, ok := strings.Cut(env, "=")
		if !ok {
			continue
		}

		if strings.HasPrefix(key, "AWS_") {
			cmd.Env = append(cmd.Env, env)
		}
	}
}

func injectKubernetesServiceAccount(ctx context.Context, cmd *osExec.Cmd) {
	ctx.Logger.V(3).Infof("Injecting Kubernetes service account")
	for _, env := range os.Environ() {
		key, _, ok := strings.Cut(env, "=")
		if !ok {
			continue
		}
		if lo.Contains(kubeEnvVars, key) {
			cmd.Env = append(cmd.Env, env)
		}
	}
}

// SetupConnections creates the necessary credential files and injects env vars into the cmd
func SetupConnection(ctx context.Context, connections ExecConnections, cmd *osExec.Cmd) (*ConnectionSetupResult, error) {
	var output ConnectionSetupResult
	var cleaners []func() error

	if lo.FromPtr(connections.FromConfigItem) != "" {
		configId := lo.FromPtr(connections.FromConfigItem)
		output.Sources = append(output.Sources, fmt.Sprintf("fromConfigItem: %s", configId))
		if err := uuid.Validate(configId); err != nil {
			return nil, fmt.Errorf("connection.fromConfigItem is not a valid uuid: %s", configId)
		}

		var scraperNamespace string
		var scraperSpec map[string]any

		{

			var scrapeConfig, err = models.FindScraperByConfigId(ctx.DB(), configId)
			if err != nil {
				return nil, fmt.Errorf("cannot setup connection from config %s: %v", configId, err)
			}
			scraperNamespace = scrapeConfig.Namespace

			ctx = ctx.WithObject(scrapeConfig)

			if err := json.Unmarshal([]byte(scrapeConfig.Spec), &scraperSpec); err != nil {
				return nil, fmt.Errorf("unable to unmarshal scrapeconfig spec (id=%s)", scrapeConfig.ID.String())
			}
		}

		if kubernetesScrapers, found, err := unstructured.NestedSlice(scraperSpec, "kubernetes"); err != nil {
			return nil, err
		} else if found {
			var kubeconfigFound bool

			for _, kscraper := range kubernetesScrapers {
				if kubeconfig, found, err := unstructured.NestedMap(kscraper.(map[string]any), "kubeconfig"); err != nil {
					return nil, err
				} else if found {
					kubeconfigFound = true

					connections.Kubernetes = &KubernetesConnection{}
					if err := runtime.DefaultUnstructuredConverter.FromUnstructured(kubeconfig, &connections.Kubernetes.Kubeconfig); err != nil {
						return nil, err
					}

					output.Sources = append(output.Sources, fmt.Sprintf("kubernetes: %s", connections.Kubernetes.String()))
					if _, _, err := connections.Kubernetes.Populate(ctx, true); err != nil {
						return nil, fmt.Errorf("failed to hydrate kubernetes connection: %w", err)
					}

					ctx = ctx.WithNamespace(scraperNamespace).WithKubernetes(connections.Kubernetes)
					break
				}
			}

			// Use local kubernetes if no external kubeconfig is found
			if !kubeconfigFound {
				injectKubernetesServiceAccount(ctx, cmd)
			}
		}
	}

	if connections.Kubernetes != nil {
		if lo.FromPtr(connections.FromConfigItem) == "" {
			// If the kubernetes connection didn't come from `fromConfigItem`, we hydrate it here
			ctx = ctx.WithKubernetes(connections.Kubernetes)
			if _, _, err := connections.Kubernetes.Populate(ctx, true); err != nil {
				return nil, fmt.Errorf("failed to hydrate kubernetes connection: %w", err)
			}
		}

		if _, pathErr := os.Stat(connections.Kubernetes.Kubeconfig.ValueStatic); pathErr == nil {
			cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", connections.Kubernetes.Kubeconfig.ValueStatic))

			if f, err := os.ReadFile(connections.Kubernetes.Kubeconfig.ValueStatic); err != nil {
				return nil, fmt.Errorf("failed to read kubeconfig: %w", err)
			} else if apiServer, err := kubernetes.GetAPIServer(f); err != nil {
				return nil, fmt.Errorf("failed to get api server: %w", err)
			} else {
				output.ApiServer = apiServer
			}
		} else {
			configPath, err := saveConfig(cmd.Dir, kubernetesConfigTemplate, connections.Kubernetes)
			if err != nil {
				return nil, fmt.Errorf("failed to store kubernetes credentials: %w", err)
			}
			cleaners = append(cleaners, func() error {
				return os.RemoveAll(filepath.Dir(configPath))
			})

			if apiServer, err := kubernetes.GetAPIServer([]byte(connections.Kubernetes.Kubeconfig.ValueStatic)); err != nil {
				return nil, fmt.Errorf("failed to get api server: %w", err)
			} else {
				output.ApiServer = apiServer
			}

			cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", configPath))
		}
	}

	if connections.ServiceAccount {
		injectKubernetesServiceAccount(ctx, cmd)
	}

	if connections.AWS != nil {
		if err := connections.AWS.Populate(ctx); err != nil {
			return nil, fmt.Errorf("failed to hydrate aws connection: %w", err)
		}

		output.Sources = append(output.Sources, fmt.Sprintf("awsConnection: %s", connections.AWS.ConnectionName))
		configPath, err := saveConfig(cmd.Dir, awsConfigTemplate, connections.AWS)
		if err != nil {
			return nil, fmt.Errorf("failed to store AWS credentials: %w", err)
		}

		cleaners = append(cleaners, func() error {
			return os.RemoveAll(filepath.Dir(configPath))
		})

		cmd.Env = append(cmd.Env, "AWS_EC2_METADATA_DISABLED=true") // https://github.com/aws/aws-cli/issues/5262#issuecomment-705832151
		cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SHARED_CREDENTIALS_FILE=%s", configPath))
		if connections.AWS.Region != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_DEFAULT_REGION=%s", connections.AWS.Region))
		}
	} else if connections.EKSPodIdentity {
		injectEksPodIdentity(ctx, cmd)
	}

	if connections.Azure != nil {
		if err := connections.Azure.HydrateConnection(ctx); err != nil {
			return nil, fmt.Errorf("failed to hydrate connection %w", err)
		}
		output.Sources = append(output.Sources, fmt.Sprintf("azureConnection: %s", connections.Azure.ConnectionName))

		// login with service principal
		runCmd := osExec.Command("az", "login", "--service-principal", "--username", connections.Azure.ClientID.ValueStatic, "--password", connections.Azure.ClientSecret.ValueStatic, "--tenant", connections.Azure.TenantID)
		if err := runCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to login: %w", err)
		}
	}

	if connections.GCP != nil {
		if err := connections.GCP.HydrateConnection(ctx); err != nil {
			return nil, fmt.Errorf("failed to hydrate connection %w", err)
		}

		output.Sources = append(output.Sources, fmt.Sprintf("gcpConnection: %s", connections.GCP.ConnectionName))

		configPath, err := saveConfig(cmd.Dir, gcloudConfigTemplate, connections.GCP)
		if err != nil {
			return nil, fmt.Errorf("failed to store gcloud credentials: %w", err)
		}

		cleaners = append(cleaners, func() error {
			return os.RemoveAll(filepath.Dir(configPath))
		})

		// to configure gcloud CLI to use the service account specified in GOOGLE_APPLICATION_CREDENTIALS,
		// we need to explicitly activate it
		runCmd := osExec.Command("gcloud", "auth", "activate-service-account", "--key-file", configPath)
		if err := runCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to activate GCP service account: %w", err)
		}

		cmd.Env = append(cmd.Env, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", configPath))
	}

	output.EnvVars = cmd.Env

	output.Cleanup = func() error {
		var errorList []error
		for _, c := range cleaners {
			if err := c(); err != nil {
				errorList = append(errorList, err)
			}
		}

		if len(errorList) > 0 {
			return fmt.Errorf("failed to cleanup: %v", errorList)
		}

		return nil
	}

	return &output, nil
}
