package FixDNS

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"time"
	"log"
	"strings"
	"errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type Auth []ssh.AuthMethod

//Returns ssh signer from private key file
func GetSigner(KeyPath string, KeyPass string) (ssh.Signer, error) {

	var (
		err    error
		signer ssh.Signer
	)

	privateKey, err := ioutil.ReadFile(KeyPath)

	if err != nil {

		return nil, err

	} else if KeyPass != "" {

		signer, err = ssh.ParsePrivateKeyWithPassphrase(privateKey, []byte(KeyPass))

	} else {

		signer, err = ssh.ParsePrivateKey(privateKey)
	}

	return signer, err
}

//Auth method from private key with or without passphrase
func Key(KeyPath string, KeyPass string) (Auth,  error) {

	signer, err := GetSigner(KeyPath, KeyPass)

	if err != nil {
		return nil, err
	}

	return Auth{
		ssh.PublicKeys(signer),
	}, nil
}

//Returns default user knows hosts file
func DefaultKnownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/.ssh/known_hosts", home), err
}

func KnownHosts(file string) (ssh.HostKeyCallback, error) {
	// hostKeyCallback, err := knownhosts.New("/home/mera/.ssh/known_hosts")
	// if err != nil {
	// 	log.Fatal("could not create hostkeycallback function: ", err)
	// }
	return knownhosts.New(file)
}

//Checks is host in known hosts file.
func CheckKnownHost(host string, remote net.Addr, key ssh.PublicKey, knownFile string) (found bool, err error) {

	var keyErr *knownhosts.KeyError

	if knownFile == "" {
		path, err := DefaultKnownHostsPath()
		if err != nil {
			return false, err
		}

		knownFile = path
	}

	callback, err := KnownHosts(knownFile)

	if err != nil {
		return false, err
	}

	err = callback(host, remote, key)

	if err == nil {
		return true, nil
	}

	if errors.As(err, &keyErr) && len(keyErr.Want) > 0 {
		return true, keyErr
	}

	if err != nil {
		return false, err
	}

	return false, nil
}

//Add a host to known hosts file
func AddKnownHost(host string, remote net.Addr, key ssh.PublicKey, knownFile string) (err error) {

	if knownFile == "" {
		path, err := DefaultKnownHostsPath()
		if err != nil {
			return err
		}

		knownFile = path
	}

	f, err := os.OpenFile(knownFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	defer f.Close()

	remoteNormalized := knownhosts.Normalize(remote.String())
	hostNormalized := knownhosts.Normalize(host)
	addresses := []string{remoteNormalized}

	if hostNormalized != remoteNormalized {
		addresses = append(addresses, hostNormalized)
	}

	_, err = f.WriteString(knownhosts.Line(addresses, key) + "\n")

	return err
}

//Simplification for CheckKnownHost function
func VerifyHost(host string, remote net.Addr, key ssh.PublicKey) error {


	hostFound, err := CheckKnownHost(host, remote, key, "")


	if hostFound && err != nil {

		return err
	}

	if hostFound && err == nil {

		return nil
	}


	return AddKnownHost(host, remote, key, "")
}

//Save error connection to file
func AddAccessDeniedHostToLogFile(PathToLogFile string, ip string, ErrorMassage error) (error){
	fopen, err := os.OpenFile(PathToLogFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil{
		return err
	}	
	if _, err := fopen.WriteString("\n\nCannot connect to " + ip + " exit with error:\n" + ErrorMassage.Error() + "\n\n"); err != nil {
		return err
	}
	
	return nil
}

type Client struct {
	*ssh.Client
	Config *Config
}

type Config struct {
    User    	string
    Address     string
    Port   		int
    Auth        Auth
    Timeout     time.Duration
    Callback    ssh.HostKeyCallback
}

var DefaultTimeout = 10 * time.Second

func NewSSHConnectionStart(User string, Address string, Port int, KeyPath string, KeyPass string) (client *Client, err error){
	var sshconfig *Config
	// callback, err := KnownHosts(DefaultKnownHostsPath())

	if err != nil {
		return
	}
	auth, err := Key(KeyPath, KeyPass)
	if err != nil {
		log.Fatal(err)
	}
    sshconfig = new(Config)
    sshconfig.User = User
    sshconfig.Address = Address
    sshconfig.Port = Port
    // Config.KeyPath = KeyPath
    // Config.KeyPass = KeyPass
    sshconfig.Auth = auth
    sshconfig.Timeout = DefaultTimeout
    sshconfig.Callback = VerifyHost

    client, err = NewConn(sshconfig)

    return
}

// Starts a client connection to SSH server based on config.
func Dial(proto string, config *Config) (*ssh.Client, error) {
	return ssh.Dial(proto, net.JoinHostPort(config.Address, fmt.Sprint(config.Port)), &ssh.ClientConfig{
		User:            config.User,
		Auth:            config.Auth,
		Timeout:         config.Timeout,
		HostKeyCallback: config.Callback,
	})
}

func NewConn(config *Config) (client *Client, err error) {
	client = &Client{
		Config: config,
	}

	client.Client, err = Dial("tcp", config)	
	return
}

func (client Client) Close() error {
	return client.Client.Close()
}

//Starts a new SSH session and runs the cmd
func (client Client) RunCommand(cmd string) (string, error) {

	var (
		err  error
		sess *ssh.Session
	)

	if sess, err = client.NewSession(); err != nil {
		return "", err
	}

	defer sess.Close()
	output, err := sess.CombinedOutput(cmd)
	return strings.ReplaceAll(string(output), "\n", ""), err
}