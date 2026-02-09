package connection

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"net/url"
	"strings"
	"time"

	"github.com/flanksource/duty/models"
	"github.com/flanksource/duty/types"
)

// +kubebuilder:object:generate=true
type ArgoConnection struct {
	ConnectionName string `yaml:"connection,omitempty" json:"connection,omitempty"`
	// URL of the ArgoCD API server.
	URL                  string `yaml:"url,omitempty" json:"url,omitempty" template:"true"`
	types.Authentication `yaml:",inline" json:",inline"`
	// Token is an optional bearer token used for auth.
	// If not set, username/password will be used to create a session token.
	Token types.EnvVar `yaml:"token,omitempty" json:"token,omitempty"`
}

func (c *ArgoConnection) FromModel(connection models.Connection) error {
	c.ConnectionName = connection.Name
	c.URL = connection.URL

	if err := c.Username.Scan(connection.Username); err != nil {
		return fmt.Errorf("error scanning username: %w", err)
	}
	if err := c.Password.Scan(connection.Password); err != nil {
		return fmt.Errorf("error scanning password: %w", err)
	}
	if token := connection.Properties["token"]; token != "" {
		if err := c.Token.Scan(token); err != nil {
			return fmt.Errorf("error scanning token: %w", err)
		}
	}

	return nil
}

func (c ArgoConnection) ToModel() models.Connection {
	properties := make(types.JSONStringMap)
	if token := c.Token.String(); token != "" {
		properties["token"] = token
	}

	return models.Connection{
		Type:       models.ConnectionTypeArgo,
		Name:       c.ConnectionName,
		URL:        c.URL,
		Username:   c.Username.String(),
		Password:   c.Password.String(),
		Properties: properties,
	}
}

// Hydrate resolves connection references and env vars into static values.
func (c *ArgoConnection) Hydrate(ctx ConnectionContext) error {
	overrides := *c

	if c.ConnectionName != "" {
		connection, err := ctx.HydrateConnectionByURL(c.ConnectionName)
		if err != nil {
			return &ArgoConnectionValidationError{err: fmt.Errorf("could not hydrate connection[%s]: %w", c.ConnectionName, err)}
		}
		if connection == nil {
			return &ArgoConnectionValidationError{err: fmt.Errorf("connection[%s] not found", c.ConnectionName)}
		}
		if err := c.FromModel(*connection); err != nil {
			return &ArgoConnectionValidationError{err: err}
		}
	}

	c.ConnectionName = overrides.ConnectionName
	if overrides.URL != "" {
		c.URL = overrides.URL
	}
	if !overrides.Username.IsEmpty() {
		c.Username = overrides.Username
	}
	if !overrides.Password.IsEmpty() {
		c.Password = overrides.Password
	}
	if !overrides.Token.IsEmpty() {
		c.Token = overrides.Token
	}

	ns := ctx.GetNamespace()

	var err error
	c.Username.ValueStatic, err = ctx.GetEnvValueFromCache(c.Username, ns)
	if err != nil {
		return fmt.Errorf("could not get argo username from env var: %w", err)
	}

	c.Password.ValueStatic, err = ctx.GetEnvValueFromCache(c.Password, ns)
	if err != nil {
		return fmt.Errorf("could not get argo password from env var: %w", err)
	}

	c.Token.ValueStatic, err = ctx.GetEnvValueFromCache(c.Token, ns)
	if err != nil {
		return fmt.Errorf("could not get argo token from env var: %w", err)
	}

	return nil
}

// Client resolves auth details and returns an Argo API client.
// This should be called after Hydrate().
func (c *ArgoConnection) Client(ctx context.Context) (*argoClient, error) {
	if strings.TrimSpace(c.URL) == "" {
		return nil, &ArgoConnectionValidationError{err: fmt.Errorf("missing argocd api url. set url or connection")}
	}

	baseURL, err := normalizeArgoURL(c.URL)
	if err != nil {
		return nil, &ArgoConnectionValidationError{err: fmt.Errorf("invalid argocd url: %w", err)}
	}

	return newArgoClient(ctx, baseURL, c.Username.ValueStatic, c.Password.ValueStatic, c.Token.ValueStatic)
}

type ArgoConnectionValidationError struct {
	err error
}

func (e *ArgoConnectionValidationError) Error() string {
	return e.err.Error()
}

func (e *ArgoConnectionValidationError) Unwrap() error {
	return e.err
}

type ArgoConnectionState struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type ArgoRepository struct {
	Name            string              `json:"name,omitempty"`
	Repo            string              `json:"repo,omitempty"`
	Labels          map[string]string   `json:"labels,omitempty"`
	ConnectionState ArgoConnectionState `json:"connectionState,omitempty"`
}

type ArgoCluster struct {
	Name            string              `json:"name,omitempty"`
	Server          string              `json:"server,omitempty"`
	Labels          map[string]string   `json:"labels,omitempty"`
	ConnectionState ArgoConnectionState `json:"connectionState,omitempty"`
}

type argoClient struct {
	baseURL    string
	token      string
	httpClient *nethttp.Client
}

func newArgoClient(ctx context.Context, baseURL, username, password, token string) (*argoClient, error) {
	var resolvedToken string

	if strings.TrimSpace(token) != "" {
		resolvedToken = normalizeBearerToken(token)
		if resolvedToken == "" {
			return nil, fmt.Errorf("authentication failed: resolved token is empty")
		}
	} else {
		user := strings.TrimSpace(username)
		pass := strings.TrimSpace(password)

		if user != "" {
			if pass == "" {
				return nil, fmt.Errorf("authentication failed: connection username is set but password is empty")
			}
			payload := map[string]string{
				"username": user,
				"password": pass,
			}
			data, err := doArgoRequest(ctx, nethttp.MethodPost, baseURL+"/api/v1/session", "", payload)
			if err != nil {
				return nil, fmt.Errorf("authentication failed: %w", err)
			}
			var response struct {
				Token string `json:"token"`
			}
			if err := json.Unmarshal(data, &response); err != nil {
				return nil, fmt.Errorf("authentication failed: failed to parse session response: %w", err)
			}
			resolvedToken = normalizeBearerToken(response.Token)
			if resolvedToken == "" {
				return nil, fmt.Errorf("authentication failed: session response did not include a token")
			}
		} else if pass != "" {
			resolvedToken = normalizeBearerToken(pass)
			if resolvedToken == "" {
				return nil, fmt.Errorf("authentication failed: connection password resolved to an empty token")
			}
		}
	}

	return &argoClient{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		token:   resolvedToken,
		httpClient: &nethttp.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *argoClient) ListRepositories(ctx context.Context) ([]ArgoRepository, error) {
	data, err := c.doRequest(ctx, nethttp.MethodGet, c.baseURL+"/api/v1/repositories?forceRefresh=true", nil)
	if err != nil {
		return nil, err
	}

	return decodeArgoList[ArgoRepository](data)
}

func (c *argoClient) ListClusters(ctx context.Context) ([]ArgoCluster, error) {
	data, err := c.doRequest(ctx, nethttp.MethodGet, c.baseURL+"/api/v1/clusters", nil)
	if err != nil {
		return nil, err
	}

	return decodeArgoList[ArgoCluster](data)
}

func (c *argoClient) doRequest(ctx context.Context, method, endpoint string, payload any) ([]byte, error) {
	return doArgoRequestWithClient(ctx, c.httpClient, method, endpoint, c.token, payload)
}

func doArgoRequest(ctx context.Context, method, endpoint, token string, payload any) ([]byte, error) {
	httpClient := &nethttp.Client{Timeout: 30 * time.Second}
	return doArgoRequestWithClient(ctx, httpClient, method, endpoint, token, payload)
}

func doArgoRequestWithClient(ctx context.Context, client *nethttp.Client, method, endpoint, token string, payload any) ([]byte, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}

	req, err := nethttp.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+normalizeBearerToken(token))
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s %s failed with %d: %s", method, endpoint, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	return respBody, nil
}

func decodeArgoList[T any](data []byte) ([]T, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") {
		var list []T
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, err
		}
		return list, nil
	}

	var wrapped struct {
		Items []T `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Items == nil {
		return []T{}, nil
	}
	return wrapped.Items, nil
}

func normalizeArgoURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("url must include scheme and host: %s", raw)
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

func normalizeBearerToken(token string) string {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = token[7:]
	}
	return strings.TrimSpace(token)
}
