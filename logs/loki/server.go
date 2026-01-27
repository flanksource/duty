package loki

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/flanksource/clicky"
	"github.com/flanksource/clicky/exec"
	"github.com/flanksource/commons/logger"
	"github.com/flanksource/deps"
)

type ServerConfig struct {
	Port     int
	DataPath string
	BinDir   string
}

func (c *ServerConfig) setDefaults() {
	if c.Port == 0 {
		c.Port = 3100
	}
	if c.BinDir == "" {
		c.BinDir = ".bin"
	}
	if c.DataPath == "" {
		c.DataPath = ".loki"
	}
}

func (c ServerConfig) URL() string {
	return fmt.Sprintf("http://localhost:%d", c.Port)
}

type Server struct {
	config   ServerConfig
	process  *exec.Process
	binPath  string
	dataPath string
}

func NewServer(config ServerConfig) *Server {
	config.setDefaults()
	return &Server{config: config}
}

func (s *Server) Start() error {
	res, err := deps.InstallWithContext(context.Background(), "loki", "any", deps.WithBinDir(s.config.BinDir))
	if err != nil {
		return fmt.Errorf("failed to install loki: %w", err)
	}

	s.binPath = filepath.Join(res.BinDir, "loki")
	s.dataPath = s.config.DataPath

	_ = os.MkdirAll(s.dataPath, 0755)

	configPath := filepath.Join(s.dataPath, "loki-config.yaml")
	if err := os.WriteFile(configPath, []byte(s.generateConfig()), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	logger.Infof("Starting %s -config-file %s (data:%s)", s.binPath, configPath, s.dataPath)
	s.process = clicky.Exec(s.binPath, "-config.file="+configPath)
	if err := s.process.Start(); err != nil {
		return fmt.Errorf("failed to start loki: %w", err)
	}

	return s.waitForReady(30 * time.Second)
}

func (s *Server) Stop() error {
	if s.process != nil {
		if err := s.process.Stop(); err != nil {
			logger.Warnf("failed to stop loki process: %v", err)
		}
	}

	return nil
}

func (s *Server) URL() string {
	return s.config.URL()
}

type LogEntry struct {
	Timestamp int64
	Message   string
}

type LogStream struct {
	Labels  map[string]string
	Entries []LogEntry
}

func (ls LogStream) toLokiFormat() map[string]any {
	values := make([][]string, len(ls.Entries))
	for i, entry := range ls.Entries {
		values[i] = []string{fmt.Sprintf("%d", entry.Timestamp), entry.Message}
	}
	return map[string]any{
		"stream": ls.Labels,
		"values": values,
	}
}

func (s *Server) UploadLogs(streams []LogStream, extraLabels map[string]string) error {
	for i := range streams {
		if streams[i].Labels == nil {
			streams[i].Labels = make(map[string]string)
		}
		maps.Copy(streams[i].Labels, extraLabels)
	}

	lokiStreams := make([]map[string]any, len(streams))
	for i, stream := range streams {
		lokiStreams[i] = stream.toLokiFormat()
	}

	logData := map[string]any{"streams": lokiStreams}
	jsonData, err := json.Marshal(logData)
	if err != nil {
		return fmt.Errorf("failed to marshal log data: %w", err)
	}

	resp, err := http.Post(s.URL()+"/loki/api/v1/push", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to push logs to loki: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to push logs, status code: %d", resp.StatusCode)
	}

	time.Sleep(2 * time.Second)
	return nil
}

func (s *Server) waitForReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(s.URL() + "/ready")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("loki did not become ready within %s", timeout)
}

func (s *Server) generateConfig() string {
	return fmt.Sprintf(`auth_enabled: false

server:
  http_listen_port: %d
  grpc_listen_port: 0

common:
  instance_addr: 127.0.0.1
  path_prefix: %s
  storage:
    filesystem:
      chunks_directory: %s/chunks
      rules_directory: %s/rules
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory

query_range:
  results_cache:
    cache:
      embedded_cache:
        enabled: true
        max_size_mb: 100

schema_config:
  configs:
    - from: 2020-10-24
      store: tsdb
      object_store: filesystem
      schema: v13
      index:
        prefix: index_
        period: 24h

storage_config:
  filesystem:
    directory: %s/storage

limits_config:
  allow_structured_metadata: true

ruler:
  alertmanager_url: http://localhost:9093
`, s.config.Port, s.dataPath, s.dataPath, s.dataPath, s.dataPath)
}
