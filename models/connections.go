package models

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/flanksource/commons/hash"
	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

// List of all connection types
const (
	ConnectionTypeAnthropic      = "anthropic"
	ConnectionTypeArgo           = "argo"
	ConnectionTypeAWS            = "aws"
	ConnectionTypeAWSKMS         = "aws_kms"
	ConnectionTypeAzure          = "azure"
	ConnectionTypeAzureDevops    = "azure_devops"
	ConnectionTypeAzureKeyVault  = "azure_key_vault"
	ConnectionTypeDiscord        = "discord"
	ConnectionTypeDynatrace      = "dynatrace"
	ConnectionTypeElasticSearch  = "elasticsearch"
	ConnectionTypeEmail          = "email"
	ConnectionTypeFolder         = "folder"
	ConnectionTypeGCP            = "google_cloud"
	ConnectionTypeGCPKMS         = "gcp_kms"
	ConnectionTypeGCS            = "gcs"
	ConnectionTypeGemini         = "gemini"
	ConnectionTypeGenericWebhook = "generic_webhook"
	ConnectionTypeGit            = "git"
	ConnectionTypeGithub         = "github"
	ConnectionTypeGitlab         = "gitlab"
	ConnectionTypeGoogleChat     = "google_chat"
	ConnectionTypeHTTP           = "http"
	ConnectionTypeIFTTT          = "ifttt"
	ConnectionTypeJMeter         = "jmeter"
	ConnectionTypeKubernetes     = "kubernetes"
	ConnectionTypeLDAP           = "ldap"
	ConnectionTypeLoki           = "loki"
	ConnectionTypeMatrix         = "matrix"
	ConnectionTypeMattermost     = "mattermost"
	ConnectionTypeMongo          = "mongo"
	ConnectionTypeMySQL          = "mysql"
	ConnectionTypeNtfy           = "ntfy"
	ConnectionTypeOllama         = "ollama"
	ConnectionTypeOpenAI         = "openai"
	ConnectionTypeOpenSearch     = "opensearch"
	ConnectionTypeOpsGenie       = "opsgenie"
	ConnectionTypePostgres       = "postgres"
	ConnectionTypePrometheus     = "prometheus"
	ConnectionTypePushbullet     = "pushbullet"
	ConnectionTypePushover       = "pushover"
	ConnectionTypeRedis          = "redis"
	ConnectionTypeRestic         = "restic"
	ConnectionTypeRocketchat     = "rocketchat"
	ConnectionTypeS3             = "s3"
	ConnectionTypeSFTP           = "sftp"
	ConnectionTypeSlack          = "slack"
	ConnectionTypeSlackWebhook   = "slackwebhook"
	ConnectionTypeSMB            = "smb"
	ConnectionTypeSQLServer      = "sql_server"
	ConnectionTypeTeams          = "teams"
	ConnectionTypeTelegram       = "telegram"
	ConnectionTypeWebhook        = "webhook"
	ConnectionTypeWindows        = "windows"
	ConnectionTypeZulipChat      = "zulip_chat"
)

// looking for a substring that starts with a space,
// 'password=', then any non-whitespace characters,
// until an ending space
var passwordRegexp = regexp.MustCompile(`password=([^;]*)`)

var _ types.ResourceSelectable = (*Connection)(nil)

type Connection struct {
	ID          uuid.UUID           `gorm:"primaryKey;unique_index;not null;column:id;default:generate_ulid()" json:"id" faker:"uuid_hyphenated"  `
	Name        string              `gorm:"column:name" json:"name" faker:"name"  `
	Namespace   string              `gorm:"column:namespace" json:"namespace"`
	Source      string              `json:"source"`
	Type        string              `gorm:"column:type" json:"type" faker:"oneof:  postgres, mysql, aws, gcp, http" `
	URL         string              `gorm:"column:url" json:"url,omitempty" faker:"url" template:"true"`
	Username    string              `gorm:"column:username" json:"username,omitempty" faker:"username"  `
	Password    string              `gorm:"column:password" json:"password,omitempty" faker:"password"  `
	Properties  types.JSONStringMap `gorm:"column:properties" json:"properties,omitempty" faker:"-" template:"true"`
	Certificate string              `gorm:"column:certificate" json:"certificate,omitempty" faker:"-"  `
	InsecureTLS bool                `gorm:"column:insecure_tls;default:false" json:"insecure_tls,omitempty" faker:"-"  `
	CreatedAt   time.Time           `gorm:"column:created_at;default:now();<-:create" json:"created_at,omitempty" faker:"-"  `
	UpdatedAt   time.Time           `gorm:"column:updated_at;default:now()" json:"updated_at,omitempty" faker:"-"  `
	CreatedBy   *uuid.UUID          `gorm:"column:created_by" json:"created_by,omitempty" faker:"-"  `
}

func (c *Connection) GetID() string {
	return c.ID.String()
}

func (c *Connection) GetName() string {
	return c.Name
}

func (c *Connection) GetNamespace() string {
	return c.Namespace
}

func (c *Connection) GetType() string {
	return c.Type
}

func (c *Connection) GetStatus() (string, error) {
	return "", nil
}

func (c *Connection) GetHealth() (string, error) {
	return "", nil
}

func (c *Connection) GetLabelsMatcher() labels.Labels {
	return noopMatcher{}
}

func (c *Connection) GetFieldsMatcher() fields.Fields {
	return noopMatcher{}
}

func (c *Connection) SetProperty(key, value string) {
	if c.Properties == nil {
		c.Properties = make(types.JSONStringMap)
	}

	c.Properties[key] = value
}

func (c Connection) TableName() string {
	return "connections"
}

func (c Connection) PK() string {
	return c.ID.String()
}

func ConnectionFromURL(url url.URL) *Connection {
	c := &Connection{}
	if url.User != nil {
		c.Username = url.User.Username()
		c.Password, _ = url.User.Password()
	}
	url.User = nil
	c.URL = url.String()
	return c
}

func (c Connection) String() string {
	if strings.ToLower(c.Type) == ConnectionTypeAWS {
		return "AWS::" + c.Username
	}

	var connection string
	// Obfuscate passwords of the form ' password=xxxxx ' from connectionString since
	// connectionStrings are used as metric labels and we don't want to leak passwords
	// Returns the Connection string with the password replaced by '###'
	if _url, err := url.Parse(c.URL); err == nil {
		if _url.User != nil {
			_url.User = nil
			connection = _url.String()
		}
	}

	return passwordRegexp.ReplaceAllString(connection, "password=###")
}

func (c Connection) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}

func (c Connection) Merge(ctx types.GetEnvVarFromCache, from any) (*Connection, error) {
	if v, ok := from.(types.WithUsernamePassword); ok {
		username := v.GetUsername()
		if !username.IsEmpty() {
			val, err := ctx.GetEnvValueFromCache(username, "")
			if err != nil {
				return nil, err
			}
			c.Username = val
		}
		password := v.GetPassword()
		if !password.IsEmpty() {
			val, err := ctx.GetEnvValueFromCache(password, "")
			if err != nil {
				return nil, err
			}
			c.Password = val
		}
	}

	if v, ok := from.(types.WithCertificate); ok {
		cert := v.GetCertificate()
		if !cert.IsEmpty() {
			val, err := ctx.GetEnvValueFromCache(cert, "")
			if err != nil {
				return nil, err
			}
			c.Certificate = val
		}
	}

	if v, ok := from.(types.WithURL); ok {
		url := v.GetURL()
		if !url.IsEmpty() {
			val, err := ctx.GetEnvValueFromCache(url, "")
			if err != nil {
				return nil, err
			}
			c.URL = val
		}
	}

	if v, ok := from.(types.WithProperties); ok {
		if c.Properties == nil {
			c.Properties = make(types.JSONStringMap)
		}
		for k, v := range v.GetProperties() {
			c.Properties[k] = v
		}
	}

	return &c, nil

}

// AsGoGetterURL returns the connection as a url that's supported by https://github.com/hashicorp/go-getter
// Connection details are added to the url as query params
func (c Connection) AsGoGetterURL() (string, error) {
	parsedURL, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}

	var output string
	switch strings.ReplaceAll(strings.ToLower(c.Type), " ", "_") {
	case ConnectionTypeHTTP:
		if c.Username != "" || c.Password != "" {
			parsedURL.User = url.UserPassword(c.Username, c.Password)
		}

		output = parsedURL.String()

	case ConnectionTypeGit:
		q := parsedURL.Query()

		if c.Username != "" || c.Password != "" {
			parsedURL.User = url.UserPassword(c.Username, c.Password)
		}

		if c.Certificate != "" {
			q.Set("sshkey", base64.URLEncoding.EncodeToString([]byte(c.Certificate)))
		}

		if v, ok := c.Properties["ref"]; ok && v != "" {
			q.Set("ref", v)
		}

		if v, ok := c.Properties["depth"]; ok && v != "" {
			q.Set("depth", v)
		}

		parsedURL.RawQuery = q.Encode()
		output = parsedURL.String()
		if !strings.HasPrefix(output, "git::") {
			output = "git::" + output
		}

	case ConnectionTypeAWS:
		q := parsedURL.Query()
		q.Set("aws_access_key_id", c.Username)
		q.Set("aws_access_key_secret", c.Password)

		if v, ok := c.Properties["profile"]; ok && v != "" {
			q.Set("aws_profile", v)
		}

		if v, ok := c.Properties["region"]; ok && v != "" {
			q.Set("region", v)
		}

		// For S3
		if v, ok := c.Properties["version"]; ok && v != "" {
			q.Set("version", v)
		}

		parsedURL.RawQuery = q.Encode()
		output = parsedURL.String()
	}

	return output, nil
}

// AsEnv generates environment variables and a configuration file content based on the connection type.
func (c Connection) AsEnv(ctx context.Context) EnvPrep {
	var envPrep = EnvPrep{
		Files: make(map[string]bytes.Buffer),
	}

	switch strings.ReplaceAll(strings.ToLower(c.Type), " ", "_") {
	case ConnectionTypeAWS:
		envPrep.Env = append(envPrep.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.Username))
		envPrep.Env = append(envPrep.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.Password))

		credentialFilePath := filepath.Join(".creds", "aws", fmt.Sprintf("cred-%d", rand.Intn(100000000)))
		if p, err := hash.JSONMD5Hash(c); err == nil {
			credentialFilePath = filepath.Join(".creds", "aws", p)
		}

		var credentialFile bytes.Buffer
		credentialFile.WriteString("[default]\n")
		credentialFile.WriteString(fmt.Sprintf("aws_access_key_id = %s\n", c.Username))
		credentialFile.WriteString(fmt.Sprintf("aws_secret_access_key = %s\n", c.Password))

		if v, ok := c.Properties["profile"]; ok && v != "" {
			envPrep.Env = append(envPrep.Env, fmt.Sprintf("AWS_DEFAULT_PROFILE=%s", v))
		}

		if v, ok := c.Properties["region"]; ok && v != "" {
			envPrep.Env = append(envPrep.Env, fmt.Sprintf("AWS_DEFAULT_REGION=%s", v))

			credentialFile.WriteString(fmt.Sprintf("region = %s\n", v))

			envPrep.CmdEnvs = append(envPrep.CmdEnvs, fmt.Sprintf("AWS_DEFAULT_REGION=%s", v))
		}

		envPrep.Files[credentialFilePath] = credentialFile

		envPrep.CmdEnvs = append(envPrep.CmdEnvs, "AWS_EC2_METADATA_DISABLED=true") // https://github.com/aws/aws-cli/issues/5262#issuecomment-705832151
		envPrep.CmdEnvs = append(envPrep.CmdEnvs, fmt.Sprintf("AWS_SHARED_CREDENTIALS_FILE=%s", credentialFilePath))

	case ConnectionTypeAzure:
		args := []string{"login", "--service-principal", "--username", c.Username, "--password", c.Password}
		if v, ok := c.Properties["tenant"]; ok && v != "" {
			args = append(args, "--tenant")
			args = append(args, v)
		}

		// login with service principal
		envPrep.PreRuns = append(envPrep.PreRuns, exec.CommandContext(ctx, "az", args...))

	case ConnectionTypeGCP:
		var credentialFile bytes.Buffer
		credentialFile.WriteString(c.Certificate)

		credentialFilePath := filepath.Join(".creds", "gcp", fmt.Sprintf("cred-%d", rand.Intn(100000000)))
		if p, err := hash.JSONMD5Hash(c); err == nil {
			credentialFilePath = filepath.Join(".creds", "gcp", p)
		}

		// to configure gcloud CLI to use the service account specified in GOOGLE_APPLICATION_CREDENTIALS,
		// we need to explicitly activate it
		envPrep.PreRuns = append(envPrep.PreRuns, exec.CommandContext(ctx, "gcloud", "auth", "activate-service-account", "--key-file", credentialFilePath))
		envPrep.Files[credentialFilePath] = credentialFile

		envPrep.CmdEnvs = append(envPrep.CmdEnvs, fmt.Sprintf("GOOGLE_APPLICATION_CREDENTIALS=%s", credentialFilePath))
	}

	return envPrep
}

type EnvPrep struct {
	// Env is the connection credentials in environment variables
	Env []string

	// CmdEnvs is a list of env vars that will be passed to the command
	CmdEnvs []string

	// List of commands that need to be run before the actual command.
	// These commands will setup the connection.
	PreRuns []*exec.Cmd

	// File contains the content of the configuration file based on the connection
	Files map[string]bytes.Buffer
}

// Inject creates the config file & injects the necessary environment variable into the command
func (c *EnvPrep) Inject(ctx context.Context, cmd *exec.Cmd) ([]*exec.Cmd, error) {
	for path, file := range c.Files {
		if err := saveConfig(file.Bytes(), path); err != nil {
			return nil, fmt.Errorf("error saving config to %s: %w", path, err)
		}
	}

	cmd.Env = append(cmd.Env, c.CmdEnvs...)

	return c.PreRuns, nil
}

func saveConfig(content []byte, absPath string) error {
	if err := os.MkdirAll(filepath.Dir(absPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create base directory for config: %w", err)
	}

	file, err := os.Create(absPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(content)
	return err
}
