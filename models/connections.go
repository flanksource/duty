package models

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/flanksource/duty/types"
	"github.com/google/uuid"
)

// List of all connection types
const (
	ConnectionTypeAWS            = "AWS"
	ConnectionTypeAzure          = "Azure"
	ConnectionTypeAzureDevops    = "Azure Devops"
	ConnectionTypeDiscord        = "Discord"
	ConnectionTypeDynatrace      = "Dynatrace"
	ConnectionTypeElasticSearch  = "ElasticSearch"
	ConnectionTypeEmail          = "Email"
	ConnectionTypeGCP            = "Google Cloud"
	ConnectionTypeGenericWebhook = "Generic Webhook"
	ConnectionTypeGit            = "Git"
	ConnectionTypeGithub         = "Github"
	ConnectionTypeGoogleChat     = "Google Chat"
	ConnectionTypeHTTP           = "HTTP"
	ConnectionTypeIFTTT          = "IFTTT"
	ConnectionTypeJMeter         = "JMeter"
	ConnectionTypeKubernetes     = "Kubernetes"
	ConnectionTypeLDAP           = "LDAP"
	ConnectionTypeMatrix         = "Matrix"
	ConnectionTypeMattermost     = "Mattermost"
	ConnectionTypeMongo          = "Mongo"
	ConnectionTypeMySQL          = "MySQL"
	ConnectionTypeNtfy           = "Ntfy"
	ConnectionTypeOpsGenie       = "OpsGenie"
	ConnectionTypePostgres       = "Postgres"
	ConnectionTypePrometheus     = "Prometheus"
	ConnectionTypePushbullet     = "Pushbullet"
	ConnectionTypePushover       = "Pushover"
	ConnectionTypeRedis          = "Redis"
	ConnectionTypeRestic         = "Restic"
	ConnectionTypeRocketchat     = "Rocketchat"
	ConnectionTypeSFTP           = "SFTP"
	ConnectionTypeSlack          = "Slack"
	ConnectionTypeSlackWebhook   = "SlackWebhook"
	ConnectionTypeSMB            = "SMB"
	ConnectionTypeSQLServer      = "SQL Server"
	ConnectionTypeTeams          = "Teams"
	ConnectionTypeTelegram       = "Telegram"
	ConnectionTypeWebhook        = "Webhook"
	ConnectionTypeWindows        = "Windows"
	ConnectionTypeZulipChat      = "Zulip Chat"
)

type Connection struct {
	ID          uuid.UUID           `gorm:"primaryKey;unique_index;not null;column:id" json:"id" faker:"uuid_hyphenated"  `
	Name        string              `gorm:"column:name" json:"name" faker:"name"  `
	Type        string              `gorm:"column:type" json:"type" faker:"oneof:  postgres, mysql, aws, gcp, http" `
	URL         string              `gorm:"column:url" json:"url,omitempty" faker:"url" template:"true"`
	Username    string              `gorm:"column:username" json:"username,omitempty" faker:"username"  `
	Password    string              `gorm:"column:password" json:"password,omitempty" faker:"password"  `
	Properties  types.JSONStringMap `gorm:"column:properties" json:"properties,omitempty" faker:"-"  `
	Certificate string              `gorm:"column:certificate" json:"certificate,omitempty" faker:"-"  `
	InsecureTLS bool                `gorm:"column:insecure_tls;default:false" json:"insecure_tls,omitempty" faker:"-"  `
	CreatedAt   time.Time           `gorm:"column:created_at;default:now()" json:"created_at,omitempty" faker:"-"  `
	UpdatedAt   time.Time           `gorm:"column:updated_at;default:now()" json:"updated_at,omitempty" faker:"-"  `
	CreatedBy   *uuid.UUID          `gorm:"column:created_by" json:"created_by,omitempty" faker:"-"  `
}

func (c Connection) String() string {
	if c.Type == "aws" {
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
	//looking for a substring that starts with a space,
	//'password=', then any non-whitespace characters,
	//until an ending space
	re := regexp.MustCompile(`password=([^;]*)`)
	return re.ReplaceAllString(connection, "password=###")
}

func (c Connection) AsMap(removeFields ...string) map[string]any {
	return asMap(c, removeFields...)
}

// AsGoGetterURL returns the connection as a url that's supported by https://github.com/hashicorp/go-getter
// Connection details are added to the url as query params
func (c Connection) AsGoGetterURL() (string, error) {
	parsedURL, err := url.Parse(c.URL)
	if err != nil {
		return "", err
	}

	var output string
	switch c.Type {
	case ConnectionTypeHTTP:
		if c.Username != "" || c.Password != "" {
			parsedURL.User = url.UserPassword(c.Username, c.Password)
		}

		output = parsedURL.String()

	case ConnectionTypeGit:
		q := parsedURL.Query()
		q.Set("sshkey", c.Certificate)

		if v, ok := c.Properties["ref"]; ok {
			q.Set("ref", v)
		}

		if v, ok := c.Properties["depth"]; ok {
			q.Set("depth", v)
		}

		parsedURL.RawQuery = q.Encode()
		output = parsedURL.String()

	case ConnectionTypeAWS:
		q := parsedURL.Query()
		q.Set("aws_access_key_id", c.Username)
		q.Set("aws_access_key_secret", c.Password)

		if v, ok := c.Properties["profile"]; ok {
			q.Set("aws_profile", v)
		}

		if v, ok := c.Properties["region"]; ok {
			q.Set("region", v)
		}

		// For S3
		if v, ok := c.Properties["version"]; ok {
			q.Set("version", v)
		}

		parsedURL.RawQuery = q.Encode()
		output = parsedURL.String()
	}

	return output, nil
}

// AsEnv generates environment variables and a configuration file content based on the connection type.
func (c Connection) AsEnv() ([]string, string) {
	var (
		envs []string
		file strings.Builder
	)

	switch c.Type {
	case ConnectionTypeAWS:
		envs = append(envs, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.Username))
		envs = append(envs, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.Password))

		file.WriteString("[default]\n")
		file.WriteString(fmt.Sprintf("aws_access_key_id = %s\n", c.Username))
		file.WriteString(fmt.Sprintf("aws_secret_access_key = %s\n", c.Password))

		if v, ok := c.Properties["profile"]; ok {
			envs = append(envs, fmt.Sprintf("AWS_DEFAULT_PROFILE=%s", v))
		}

		if v, ok := c.Properties["region"]; ok {
			envs = append(envs, fmt.Sprintf("AWS_DEFAULT_REGION=%s", v))
			file.WriteString(fmt.Sprintf("region = %s\n", v))
		}

	case ConnectionTypeGCP:
		file.WriteString(c.Certificate)
	}

	return envs, file.String()
}
