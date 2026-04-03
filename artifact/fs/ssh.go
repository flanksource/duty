package fs

import (
	gocontext "context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sftpClient "github.com/flanksource/duty/artifact/clients/sftp"
	"github.com/flanksource/duty/artifact"
	"github.com/pkg/sftp"
)

type sshFS struct {
	*sftp.Client
	wd string
}

type sshFileInfo struct {
	fullpath string
	fs.FileInfo
}

func (t *sshFileInfo) FullPath() string {
	return t.fullpath
}

func NewSSHFS(host, user, password string) (*sshFS, error) {
	client, err := sftpClient.SSHConnect(host, user, password)
	if err != nil {
		return nil, err
	}

	wd, err := client.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	return &sshFS{
		wd:     wd,
		Client: client,
	}, nil
}

func (t *sshFS) ReadDir(name string) ([]artifact.FileInfo, error) {
	if strings.Contains(name, "*") {
		return t.ReadDirGlob(name)
	}

	files, err := t.Client.ReadDir(name)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.FileInfo, 0, len(files))
	for _, file := range files {
		base := name
		if !strings.HasPrefix(name, "/") {
			base = filepath.Join(t.wd, name)
		}
		output = append(output, &sshFileInfo{FileInfo: file, fullpath: filepath.Join(base, file.Name())})
	}

	return output, nil
}

func (t *sshFS) ReadDirGlob(name string) ([]artifact.FileInfo, error) {
	entries, err := t.Glob(name)
	if err != nil {
		return nil, err
	}

	output := make([]artifact.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := t.Stat(entry)
		if err != nil {
			return nil, err
		}
		output = append(output, &sshFileInfo{FileInfo: info, fullpath: entry})
	}

	return output, nil
}

func (s *sshFS) Read(_ gocontext.Context, path string) (io.ReadCloser, error) {
	return s.Open(path)
}

func (s *sshFS) Close() error {
	return s.Client.Close()
}

func (s *sshFS) Write(_ gocontext.Context, path string, data io.Reader) (os.FileInfo, error) {
	dir := filepath.Dir(path)
	if err := s.MkdirAll(dir); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	f, err := s.Create(path)
	if err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}
	defer f.Close()

	if _, err = io.Copy(f, data); err != nil {
		return nil, fmt.Errorf("error writing to file: %w", err)
	}

	return f.Stat()
}
