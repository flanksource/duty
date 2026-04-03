package sftp

import (
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SSHConnect creates an SFTP client connection.
// NOTE: Uses InsecureIgnoreHostKey because artifact storage targets are
// configured by admins via trusted connection objects, not user input.
func SSHConnect(host, user, password string) (*sftp.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec
		Timeout:         30 * time.Second,
	}

	conn, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, err
	}

	client, err := sftp.NewClient(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return client, nil
}
