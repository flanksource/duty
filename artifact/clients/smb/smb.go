package smb

import (
	"net"

	"github.com/flanksource/duty/types"
	"github.com/hirochachacha/go-smb2"
)

type SMBSession struct {
	net.Conn
	*smb2.Session
	*smb2.Share
}

func SMBConnect(server string, port, share string, auth types.Authentication) (*SMBSession, error) {
	var err error
	var smb *SMBSession
	server = server + ":" + port
	conn, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}
	smb = &SMBSession{
		Conn: conn,
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     auth.GetUsername(),
			Password: auth.GetPassword(),
			Domain:   auth.GetDomain(),
		},
	}

	s, err := d.Dial(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	smb.Session = s
	fs, err := s.Mount(share)
	if err != nil {
		_ = s.Logoff()
		conn.Close()
		return nil, err
	}

	smb.Share = fs

	return smb, err
}

func (s *SMBSession) Close() error {
	if s.Conn != nil {
		_ = s.Conn.Close()
	}
	if s.Session != nil {
		_ = s.Logoff()
	}
	if s.Share != nil {
		_ = s.Umount()
	}

	return nil
}
