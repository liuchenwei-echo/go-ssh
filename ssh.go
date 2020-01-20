package ssh

import (
	"errors"
	"go-ssh/utils"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/proxy"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	PasswordAuth = "password"
	KeyAuth      = "key"
)

// 获取SSH连接
func GetSshShell(serverConfig *ServerConfig, globalConfig *GlobalConfig) error {
	// 获取ssh客户端
	client, err := GetSshClient(serverConfig)
	if err != nil {
		log.Fatalf("unable to create client: %s", err)
	}
	defer client.Close()
	// 创建ssh session
	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %s", err)
	}
	defer session.Close()
	// 创建fd
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		return errors.New("unable to create fd:" + err.Error())
	}
	defer terminal.Restore(fd, oldState)
	// 保持链接
	stopKeepAliveLoop := keepAliveLoop(session, globalConfig)
	defer close(stopKeepAliveLoop)
	// 重定向标准输出流
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	// 终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	// 终端大小
	termWidth, termHeight, _ := terminal.GetSize(fd)
	termType := os.Getenv("TERM")
	if termType == "" {
		termType = "xterm-256color"
	}
	if err := session.RequestPty(termType, termHeight, termWidth, modes); err != nil {
		return errors.New("unable to request pty:" + err.Error())
	}
	// 监听窗口变化
	listenWindowSizeChange(session, fd)
	// 开启shell模式
	err = session.Shell()
	if err != nil {
		return errors.New("unable to exec shell:" + err.Error())
	}
	_ = session.Wait()
	return nil
}

// 获取SSH连接客户端
func GetSshClient(serverConfig *ServerConfig) (*ssh.Client, error) {
	// 授权方式
	var authMethod []ssh.AuthMethod
	if serverConfig.AuthMethod == PasswordAuth {
		authMethod = append(authMethod, ssh.Password(serverConfig.Password))
	} else if serverConfig.AuthMethod == KeyAuth {
		// 获取秘钥授权
		publicAuth, err := GetSshPublicKeyAuth(serverConfig)
		if err != nil {
			return nil, err
		}
		authMethod = append(authMethod, publicAuth)
	}

	clientConfig := &ssh.ClientConfig{
		User: serverConfig.User,
		Auth: authMethod,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	// 默认端口为22
	if serverConfig.Port == 0 {
		serverConfig.Port = 22
	}
	// ssh address
	address := serverConfig.Host + ":" + strconv.Itoa(serverConfig.Port)
	// ssh proxy
	var client *ssh.Client
	var err error
	if len(serverConfig.Proxy) > 0 && strings.HasPrefix(serverConfig.Proxy, "socks5://") {
		var dialer proxy.Dialer
		proxyUrl := string(serverConfig.Proxy[9:])
		dialer, err = proxy.SOCKS5("tcp", proxyUrl, nil, proxy.Direct)
		if err == nil {
			var conn net.Conn
			conn, err = dialer.Dial("tcp", address)
			if err == nil {
				c, newchan, reqs, err := ssh.NewClientConn(conn, address, clientConfig)
				if err == nil {
					client, err = ssh.NewClient(c, newchan, reqs), nil
				}
			}
		}
		if err != nil {
			log.Fatalf("unable to create proxy: %s", err.Error())
		}
	}

	// 直连
	if client == nil {
		client, err = ssh.Dial("tcp", address, clientConfig)
	}
	if err != nil {
		return nil, err
	}
	return client, nil
}

// 获取SSH秘钥授权
func GetSshPublicKeyAuth(config *ServerConfig) (ssh.AuthMethod, error) {
	if config.Key == "" {
		config.Key = "~/.ssh/id_rsa"
	}
	keyPath, err := utils.ParseAbsolutePath(config.Key)
	if err != nil {
		return nil, err
	}
	fileBytes, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}
	var signer ssh.Signer
	if config.Password == "" {
		signer, err = ssh.ParsePrivateKey(fileBytes)
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(fileBytes, []byte(config.Password))
	}
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// 监听终端窗口变化
func listenWindowSizeChange(session *ssh.Session, fd int) {
	go func() {
		sigwinchCh := make(chan os.Signal, 1)
		defer close(sigwinchCh)
		signal.Notify(sigwinchCh, syscall.SIGWINCH)
		termWidth, termHeight, err := terminal.GetSize(fd)
		if err != nil {
			panic(err)
		}
		for {
			select {
			// 阻塞读取
			case sigwinch := <-sigwinchCh:
				if sigwinch == nil {
					return
				}
				currTermWidth, currTermHeight, err := terminal.GetSize(fd)
				if err != nil {
					continue
				}
				// 判断一下窗口尺寸是否有改变
				if currTermHeight == termHeight && currTermWidth == termWidth {
					continue
				}
				// 更新远端大小
				session.WindowChange(currTermHeight, currTermWidth)
				termWidth, termHeight = currTermWidth, currTermHeight
			}
		}
	}()
}

// 保持链接
func keepAliveLoop(session *ssh.Session, globalConfig *GlobalConfig) chan struct{} {
	terminate := make(chan struct{})
	go func() {
		for {
			select {
			case <-terminate:
				return
			default:
				if globalConfig != nil {
					if globalConfig.ServerAliveInterval > 0 {
						session.SendRequest("keepalive@gossh", true, nil)
						t := time.Duration(globalConfig.ServerAliveInterval)
						time.Sleep(time.Second * t)
						continue
					}
				}
				return
			}
		}
	}()
	return terminate
}
