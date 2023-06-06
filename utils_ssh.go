package zfs

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type RemoteStruct struct {
	User       string
	Host       string
	Port       string
	KeyPath    string
	Client     *ssh.Client
	Session    *ssh.Session
	KeepClient bool // keep client connection
}

// RemoteConfig is the global ssh config to be used by all zfs commands.
var RemoteConfig *RemoteStruct

// String representation of ssh config.
func (remote *RemoteStruct) String() string {
	result := "<unknown>"
	if remote.Host != "" {
		if remote.User != "" {
			result = remote.User + "@" + remote.Host
		}
		if remote.Port != "" {
			result += ":" + remote.Port
		}
	}
	return result
}

func NewRemoteConfig(remote RemoteStruct) *RemoteStruct {
	return &remote
}

// Close and clear ssh.Client connection.
func (remote *RemoteStruct) Close() error {
	err := remote.Client.Close()
	remote.Client = nil
	return err
}

/*
func publicKeyFile(keyPath string) ssh.AuthMethod {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}
	return ssh.PublicKeys(signer)
}
*/

// NewRemoteClient creates a new ssh.Client connection.
func NewRemoteClient(remote *RemoteStruct) (*ssh.Client, error) {
	// TODO: add support for OpenSSH configuration files
	// https://pkg.go.dev/golang.org/x/crypto/ssh#Config

	var err error

	currentUser, _ := user.Current()

	// Set default keypath to ~/.ssh/id_rsa
	if remote.KeyPath == "" {
		remote.KeyPath = filepath.Join(currentUser.HomeDir, "/.ssh/id_rsa")
	}

	// Read the private key file
	key, err := os.ReadFile(remote.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse the private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	hostKeyCallback, err := knownhosts.New(filepath.Join(currentUser.HomeDir, ".ssh/known_hosts"))
	if err != nil {
		return nil, err
	}

	// Create the SSH config
	config := &ssh.ClientConfig{
		User: remote.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		HostKeyCallback: hostKeyCallback,
	}

	// use default port if none specified in sshConfig
	if remote.Port == "" {
		remote.Port = "22"
	}
	host := remote.Host + ":" + remote.Port

	// Connect to the remote Host
	remote.Client, err = ssh.Dial("tcp", host, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Host: %w", err)
	}
	return remote.Client, nil
}

func (remote *RemoteStruct) NewRemoteSession() (*ssh.Session, error) {
	var err error

	if remote.KeepClient && remote.Client != nil {
		fmt.Println("Re-using existing SSH client ...")
	} else {
		remote.Client, err = NewRemoteClient(remote)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection: %w", err)
		}
	}

	// Create a new Session
	remote.Session, err = remote.Client.NewSession()
	if err != nil {
		remote.Client.Close()
		return nil, fmt.Errorf("failed to create Session: %w", err)
	}
	return remote.Session, nil
}
