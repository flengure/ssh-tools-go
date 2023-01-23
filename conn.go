package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/povsister/scp"
	"golang.org/x/crypto/ssh"
)

var UseSytemDefaultUsername = false
var DefaultUsername = "root"

var GetDefaultUsername = func() string {
	if UseSytemDefaultUsername {
		user, err := user.Current()
		if err != nil {
			return DefaultUsername
		}
		return user.Username
	}
	return DefaultUsername
}()

func wordWrap(text string, lineWidth int) string {
	wrap := make([]byte, 0, len(text)+2*len(text)/lineWidth)
	eoLine := lineWidth
	inWord := false
	for i, j := 0, 0; ; {
		r, size := utf8.DecodeRuneInString(text[i:])
		if size == 0 && r == utf8.RuneError {
			r = ' '
		}
		if unicode.IsSpace(r) {
			if inWord {
				if i >= eoLine {
					wrap = append(wrap, '\n')
					eoLine = len(wrap) + lineWidth
				} else if len(wrap) > 0 {
					wrap = append(wrap, ' ')
				}
				wrap = append(wrap, text[j:i]...)
			}
			inWord = false
		} else if !inWord {
			inWord = true
			j = i
		}
		if size == 0 && r == ' ' {
			break
		}
		i += size
	}
	return string(wrap)
}

func path_exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func ParseHostSpecToUserHost(s string) (string, string) {

	var user string
	var host string
	var port string
	var p int
	var q []string
	var r []string

	if s == "" {
		return GetDefaultUsername, "localhost:22"
	}

	q = strings.Split(s, "@")
	if len(q) == 1 {
		user = GetDefaultUsername
		host = s
	} else {
		user = q[0]
		host = strings.Join(q[1:], "")
	}

	r = strings.Split(host, ":")
	if len(r) == 1 {
		port = "22"
	} else if len(r) > 0 {
		host = strings.Join(r[:len(r)-1], "")
		port = r[len(r)-1]
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		p = 22
	}
	if p < 1 || p > 65535 {
		p = 22
	}

	return user, host + ":" + strconv.Itoa(p)

}

func generate_ssh_keys() (string, error) {

	// home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	private := filepath.Join(home, ".ssh", "id_ed25519")
	if path_exists(private) {
		return private, nil
	}
	// fmt.Println(private)

	public := private + ".pub"
	user, _ := user.Current()
	host, _ := os.Hostname()

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}

	publicKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", err
	}

	pemKey := &pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: MarshalED25519PrivateKey(privKey), // <- marshals ed25519 correctly
	}

	privateKey := pem.EncodeToMemory(pemKey)

	authorizedKey := []byte(
		fmt.Sprintf("%s %s@%s",
			strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(publicKey)), "\n"),
			user.Username,
			host,
		),
	)

	err = os.WriteFile(private, privateKey, 0600)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(public, authorizedKey, 0644)
	if err != nil {
		return "", err
	}

	return private, nil

}

type conn struct {
	ssh      *scp.Client
	host     string
	password string
	key      string
	os       string
}

func (c *conn) Connect() error {

	var err error
	user, host := ParseHostSpecToUserHost(c.host)

	// fmt.Printf("%s: %s\n", user, host)
	// fmt.Println(GetDefaultUsername)

	sk, _ := get_keys(c.key)

	sshClientConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshClientConfig.Auth = []ssh.AuthMethod{
		ssh.PublicKeys(sk.signer),
		ssh.Password(c.password),
	}

	// Connect to the remote server and perform the SSH handshake.
	//client, err := scp.NewClient(host, sshClientConfig, &scp.ClientOption{})
	c.ssh, err = scp.NewClient(host, sshClientConfig, &scp.ClientOption{})

	if err != nil {
		return err
	}

	/* Write the public key associated with the private key that made
	this successful connection to the authorized_keys file of the remote
	server.
	Will do both ~/.ssh/authorized_keys and /etc/dropbear/authorized_keys
	*/

	var sb strings.Builder
	sb.WriteString("k='" + sk.public_key + "'; ")
	sb.WriteString("for d in \"/etc/dropbear\" \"~/.ssh\"; do ")
	sb.WriteString("f=\"$d/authorized_keys\"; ")
	sb.WriteString("if [ -d \"$d\" ]; then ")
	sb.WriteString("[ -f \"$f\" ] || echo \"$k\" >> \"$f\"; ")
	sb.WriteString("grep -q \"$k\" \"$f\" || echo \"$k\" >> \"$f\"; ")
	sb.WriteString("fi; done;")

	if sk.public_key != "" {
		_ = c.run(sb.String())
	}

	os, _ := c.output("cmd /c ver || uname -a")

	if strings.Contains(strings.ToLower(os), "windows") {
		c.os = "windows"
	} else if strings.Contains(strings.ToLower(os), "linux") {
		c.os = "linux"
	} else if strings.Contains(strings.ToLower(os), "darwin") {
		c.os = "darwin"
	}
	// fmt.Println(c.os)
	// fmt.Println(c.os)

	return nil
}

func (c *conn) isConnected() bool {
	return c.ssh != nil
}

func (c *conn) get_content_scp(remotePath string) (string, error) {

	// takes a remote file path
	// returns the contents as a string
	// using scp

	if !c.isConnected() {
		err := c.Connect()
		if err != nil {
			return "", err
		}
	}

	sb := new(strings.Builder)

	err := c.ssh.CopyFromRemote(remotePath, sb, &scp.FileTransferOption{})
	if err != nil {
		return "", err
	}

	return sb.String(), nil
}

func (c *conn) get_content_ssh(remotePath string) (string, error) {

	// takes a remote file path
	// returns the contents as a string
	// using ssh mode

	if !c.isConnected() {
		err := c.Connect()
		if err != nil {
			return "", err
		}
	}

	// open a client conn
	sess, err := c.ssh.NewSession()
	if err != nil {
		return "", err
	}

	defer sess.Close()

	// run cat command
	// cat filename
	// where filename
	result, err := sess.Output(fmt.Sprintf("cat \"%s\"", remotePath))
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func (c *conn) get_content(remotePath string) (string, error) {
	result, err := c.get_content_scp(remotePath)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (c *conn) set_content_scp(text, remotePath string) error {

	// takes some text and saves it to a remote file
	// replacing the existing content
	// uses scp mode

	if !c.isConnected() {
		err := c.Connect()
		if err != nil {
			return err
		}
	}

	reader := strings.NewReader(text)

	err := c.ssh.CopyToRemote(reader, remotePath, &scp.FileTransferOption{})
	if err != nil {
		return err
	}

	return nil
}

func (c *conn) set_content_ssh(text, remotePath string) error {

	// takes some text and saves it to a remote file
	// replacing the existing content
	// uses ssh and an os specific command on the target
	// to pipe the data to a file

	if !c.isConnected() {
		err := c.Connect()
		if err != nil {
			return err
		}
	}

	// open a client conn
	sess, err := c.ssh.NewSession()
	if err != nil {
		return err
	}

	defer sess.Close()

	// stdin pipe
	w, err := sess.StdinPipe()
	if err != nil {
		return err
	}

	defer w.Close()

	// echo to remote file
	// os specific command here
	// not very portable
	err = sess.Start(fmt.Sprintf("cat > \"%s\"", remotePath))
	if err != nil {
		return err
	}

	// write the text to the pipe
	i, err := fmt.Fprintf(w, text)
	if err != nil {
		return err
	}

	// what do I do with this ?
	fmt.Println(i)
	return nil
}

func (c *conn) set_content(text, remotePath string) error {
	err := c.set_content_scp(text, remotePath)
	if err != nil {
		return err
	}

	return nil
}

func (c *conn) run(text string) error {

	var sess *ssh.Session
	var err error

	if !c.isConnected() {
		err := c.Connect()
		if err != nil {
			return err
		}
	}

	sess, err = c.ssh.NewSession()
	if err != nil {
		return err
	}

	defer sess.Close()

	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	// run command specified by text
	err = sess.Run(text)
	if err != nil {
		return err
	}

	return nil
}

func (c *conn) output(text string) (string, error) {

	var sess *ssh.Session
	var err error

	if !c.isConnected() {
		err := c.Connect()
		if err != nil {
			return "", err
		}
	}

	sess, err = c.ssh.NewSession()
	if err != nil {
		return "", err
	}

	defer sess.Close()

	// run command specified by text
	result, err := sess.Output(text)
	if err != nil {
		return "", err
	}

	return string(result), nil

}

type ssh_key struct {
	private_key_file string
	public_key_file  string
	public_key       string
	signer           ssh.Signer
}

func get_keys(s string) (ssh_key, error) {

	o := ssh_key{}

	if s == "" || !path_exists(s) {
		new_private, err := generate_ssh_keys()
		if err != nil {
			return o, err
		}
		o.private_key_file = new_private
	} else {
		o.private_key_file = s
	}
	o.public_key_file = o.private_key_file + ".pub"

	if !path_exists(o.public_key_file) {

		ssh_keygen, err := exec.LookPath("ssh-keygen")
		if err != nil {
			log.Println("Could not find ssh-keygen")
		}

		cmd := exec.Command(
			ssh_keygen, "-f", o.private_key_file, "-y", ">", o.public_key_file)

		if err := cmd.Run(); err != nil {
			log.Println("error running ssh-keygen")
		}
	}

	if path_exists(o.public_key_file) {

		f, err := os.ReadFile(o.public_key_file)
		if err != nil {
			log.Println("error reading public key file" + o.public_key_file)
		} else {
			o.public_key = strings.TrimSuffix(string(string(f)), "\n")
		}
	}

	if path_exists(o.private_key_file) {

		f, err := os.ReadFile(o.private_key_file)
		if err != nil {
			log.Println("error reading private key file" + o.private_key_file)
		} else {
			// Create the Signer for this private key.
			signer, err := ssh.ParsePrivateKey(f)
			if err != nil {
				log.Println("error parsing private key file" + o.private_key_file)
			} else {
				o.signer = signer
			}
		}
	}

	return o, nil
}
