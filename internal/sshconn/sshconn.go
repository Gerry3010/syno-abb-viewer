// Package sshconn dials the DiskStation over SSH and opens an SFTP session.
//
// Auth (key vs password) is a switch inside Dial — not an interface — because
// there are exactly two concrete paths and no third caller. Host keys are
// verified against ~/.ssh/known_hosts; an unknown host triggers a trust-on-first-use
// callback so the UI can ask the user before accepting a fingerprint.
package sshconn

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Gerry3010/syno-abb-viewer/internal/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// TrustFunc is called when a host is not yet in known_hosts. It receives the
// host and the SHA256 fingerprint and returns whether to trust and persist it.
type TrustFunc func(hostport, fingerprint string) bool

// Conn bundles the live SSH client and its SFTP session so both can be closed.
type Conn struct {
	Client *ssh.Client
	SFTP   *sftp.Client
}

// Close tears down the SFTP session and the SSH client.
func (c *Conn) Close() error {
	var first error
	if c.SFTP != nil {
		if err := c.SFTP.Close(); err != nil {
			first = err
		}
	}
	if c.Client != nil {
		if err := c.Client.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

// Dial connects using cfg and returns an SFTP-capable connection. trust may be
// nil, in which case unknown hosts are rejected.
func Dial(cfg config.Config, trust TrustFunc) (*Conn, error) {
	auth, err := authMethods(cfg)
	if err != nil {
		return nil, err
	}
	hostKey, err := hostKeyCallback(trust)
	if err != nil {
		return nil, err
	}
	clientCfg := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            auth,
		HostKeyCallback: hostKey,
		Timeout:         15 * time.Second,
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	client, err := ssh.Dial("tcp", addr, clientCfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s: %w", addr, err)
	}
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("open sftp: %w", err)
	}
	return &Conn{Client: client, SFTP: sftpClient}, nil
}

// authMethods builds the SSH auth list from the configured method. For key auth
// an encrypted key falls back to cfg.Password as the passphrase.
func authMethods(cfg config.Config) ([]ssh.AuthMethod, error) {
	switch cfg.Auth {
	case config.AuthPassword:
		if cfg.Password == "" {
			return nil, errors.New("password required")
		}
		return []ssh.AuthMethod{ssh.Password(cfg.Password)}, nil
	default: // AuthKey
		data, err := os.ReadFile(expandPath(cfg.KeyPath))
		if err != nil {
			return nil, fmt.Errorf("read key: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(data)
		if err != nil {
			var missing *ssh.PassphraseMissingError
			if errors.As(err, &missing) {
				if cfg.Password == "" {
					return nil, fmt.Errorf("key %s is passphrase-protected: enter the passphrase in the password field", cfg.KeyPath)
				}
				signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(cfg.Password))
			}
			if err != nil {
				return nil, fmt.Errorf("parse key: %w", err)
			}
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
}

// hostKeyCallback verifies against known_hosts, delegating unknown hosts to trust.
func hostKeyCallback(trust TrustFunc) (ssh.HostKeyCallback, error) {
	khPath, err := knownHostsPath()
	if err != nil {
		return nil, err
	}
	known, err := knownhosts.New(khPath)
	if err != nil {
		return nil, fmt.Errorf("read known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := known(hostname, remote, key); err == nil {
			return nil // already trusted
		} else {
			var keyErr *knownhosts.KeyError
			if !errors.As(err, &keyErr) {
				return err
			}
			if len(keyErr.Want) > 0 {
				// A different key is on file — refuse (possible MITM).
				return fmt.Errorf("host key mismatch for %s — refusing to connect", hostname)
			}
			// Unknown host: trust-on-first-use.
			fp := ssh.FingerprintSHA256(key)
			if trust == nil || !trust(hostname, fp) {
				return fmt.Errorf("host key for %s not trusted", hostname)
			}
			return appendKnownHost(khPath, hostname, key)
		}
	}, nil
}

// appendKnownHost persists a newly trusted host key.
func appendKnownHost(khPath, hostname string, key ssh.PublicKey) error {
	if err := os.MkdirAll(filepath.Dir(khPath), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(khPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
	_, err = fmt.Fprintln(f, line)
	return err
}

// knownHostsPath returns ~/.ssh/known_hosts, creating an empty file if absent so
// knownhosts.New can parse it.
func knownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			return "", err
		}
		f, err := os.OpenFile(p, os.O_CREATE, 0o600)
		if err != nil {
			return "", err
		}
		f.Close()
	}
	return p, nil
}

// expandPath resolves a leading ~ to the user's home directory.
func expandPath(p string) string {
	if p == "~" || (len(p) >= 2 && p[:2] == "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}
