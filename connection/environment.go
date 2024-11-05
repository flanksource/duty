package connection

import (
	"fmt"
	"math/rand"
	"os"
	osExec "os/exec"
	"path/filepath"

	"github.com/flanksource/commons/logger"
	"github.com/flanksource/duty/context"

	textTemplate "text/template"
)

// +kubebuilder:object:generate=true
type ExecConnections struct {
	Kubernetes *KubernetesConnection `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty"`
	AWS        *AWSConnection        `yaml:"aws,omitempty" json:"aws,omitempty"`
	GCP        *GCPConnection        `yaml:"gcp,omitempty" json:"gcp,omitempty"`
	Azure      *AzureConnection      `yaml:"azure,omitempty" json:"azure,omitempty"`
}

func saveConfig(configTemplate *textTemplate.Template, view any) (string, error) {
	dirPath := filepath.Join(".creds", fmt.Sprintf("cred-%d", rand.Intn(10000000)))
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

	kubernetesConfigTemplate = textTemplate.Must(textTemplate.New("").Parse(`{{.KubeConfig.ValueStatic}}`))
}

// SetupCConnections creates the necessary credential files and injects env vars
// into the cmd
func SetupConnection(ctx context.Context, connections ExecConnections, cmd *osExec.Cmd) (func() error, error) {
	var cleaner = func() error {
		return nil
	}

	if connections.Kubernetes != nil {
		configPath, err := saveConfig(kubernetesConfigTemplate, connections.Kubernetes)
		if err != nil {
			return nil, fmt.Errorf("failed to store kubernetes credentials: %w", err)
		}

		cleaner = func() error {
			return os.RemoveAll(filepath.Dir(configPath))
		}

		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", configPath))
	}

	if connections.AWS != nil {
		if err := connections.AWS.Populate(ctx); err != nil {
			return nil, fmt.Errorf("failed to hydrate aws connection: %w", err)
		}

		configPath, err := saveConfig(awsConfigTemplate, connections.AWS)
		if err != nil {
			return nil, fmt.Errorf("failed to store AWS credentials: %w", err)
		}

		cleaner = func() error {
			return os.RemoveAll(filepath.Dir(configPath))
		}

		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "AWS_EC2_METADATA_DISABLED=true") // https://github.com/aws/aws-cli/issues/5262#issuecomment-705832151
		cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SHARED_CREDENTIALS_FILE=%s", configPath))
		if connections.AWS.Region != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_DEFAULT_REGION=%s", connections.AWS.Region))
		}
	}

	if connections.Azure != nil {
		if err := connections.Azure.HydrateConnection(ctx); err != nil {
			return nil, fmt.Errorf("failed to hydrate connection %w", err)
		}

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

		configPath, err := saveConfig(gcloudConfigTemplate, connections.GCP)
		if err != nil {
			return nil, fmt.Errorf("failed to store gcloud credentials: %w", err)
		}

		cleaner = func() error {
			return os.RemoveAll(filepath.Dir(configPath))
		}

		// to configure gcloud CLI to use the service account specified in GOOGLE_APPLICATION_CREDENTIALS,
		// we need to explicitly activate it
		runCmd := osExec.Command("gcloud", "auth", "activate-service-account", "--key-file", configPath)
		if err := runCmd.Run(); err != nil {
			return nil, fmt.Errorf("failed to activate GCP service account: %w", err)
		}

		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", configPath))
	}

	return cleaner, nil
}
