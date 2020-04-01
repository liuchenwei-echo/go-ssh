package main

import (
	"fmt"
	"go-ssh"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

const (
	usage = `
Version:
	%s build on %s by %s

Usage:
	s [alias|ssh arguments]	ssh connect
	s <command> [arguments]

Commands:
	add <alias>	connect ssh
	edit <alias>	add ssh config
	rm <alias>	list ssh config
	list		edit ssh config
	help <alias>	for more information about that command 
`
	addUsage = `
Usage:
	s add [arguments]	add ssh config

Options:
	-h host			ssh server host
	-u user			ssh server login user
	-p password		ssh server login password
	-P port			ssh server port
	-a auth method 		ssh server login auth method: [password|key]
	-k keyPath		ssh server key path (default: ~/.ssh/id_rsa)
	-proxy proxy		only socks5 like socks5://127.0.0.1:1086
`

	editUsage = `
Usage:
	s add [arguments]	add ssh config

Options:
	-h host			ssh server host
	-u user			ssh server login user
	-p password		ssh server login password
	-P port			ssh server port
	-a auth method 		ssh server login auth method: [password|key]
	-k keyPath		ssh server key path (default: ~/.ssh/id_rsa)
	-proxy proxy		only socks5 like socks5://127.0.0.1:1086
`
)

var (
	Version = "unknown"
	BuildOn = "unknown"
	User    = "unknown"
)

func main() {

	// command
	var cmd string
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	} else {
		doHelp(nil)
		return
	}

	switch cmd {
	case "help":
		doHelp(os.Args[1:])
	case "add":
		doAdd(os.Args[1:])
	case "edit":
		doEdit(os.Args[1:])
	case "ls":
		doList(os.Args[1:])
	case "config":
		doConfig()
	case "rm":
		doRemove(os.Args[1:])
	default:
		doSSH(os.Args[1:])
	}
}

// 连接SSH
func doSSH(args []string) {
	if len(args) == 1 {
		configs := ssh.LoadConfig()
		serverConfig := configs.Servers[args[0]]
		if serverConfig != nil {
			ssh.GetSshShell(serverConfig, configs.Global)
			return
		}
	}
	// 透传
	passThrough(args)
}

// 解析参数
func ParseArguments(args []string) map[string]string {
	params := make(map[string]string, 0)
	var key string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			key = string(arg[2:])
		} else if strings.HasPrefix(arg, "-") {
			key = string(arg[1:])
		} else {
			if len(key) > 0 {
				params[key] = arg
				// clear key
				key = ""
			}
			// ignore
		}
	}
	return params
}

// 编辑配置
func doEdit(args []string) {
	if len(args) > 1 {
		configs := ssh.LoadConfig()
		// alias
		alias := args[1]
		serverConfig := configs.Servers[alias]
		if serverConfig == nil {
			log.Fatalf("no such ssh server config alias: %s", alias)
		}
		// 参数解析
		params := ParseArguments(args[2:])
		var port int
		if len(params["P"]) > 0 {
			port, _ = strconv.Atoi(params["P"])
			serverConfig.Port = port
		}
		if len(params["h"]) > 0 {
			serverConfig.Host = params["h"]
		}
		if len(params["u"]) > 0 {
			serverConfig.User = params["u"]
		}
		if len(params["a"]) > 0 {
			serverConfig.AuthMethod = params["a"]
		}
		if len(params["p"]) > 0 {
			serverConfig.Password = params["p"]
		}
		if len(params["k"]) > 0 {
			serverConfig.Key = params["k"]
		}
		if len(params["proxy"]) > 0 {
			serverConfig.Proxy = params["proxy"]
		}
		// 检查
		serverConfig.Check()
		// 添加配置
		err := configs.AddServerConfig(alias, serverConfig)
		if err != nil {
			log.Fatalf("edit ssh server config error: %s", err.Error())
		}
	} else {
		log.Fatalf("can't edit ssh server config without arguments")
	}
}

// 添加配置
func doAdd(args []string) {
	if len(args) > 1 {
		configs := ssh.LoadConfig()

		alias := args[1]
		params := ParseArguments(args[2:])

		var port int
		if len(params["P"]) > 0 {
			port, _ = strconv.Atoi(params["P"])
		}
		serverConfig := &ssh.ServerConfig{
			Host:       params["h"],
			User:       params["u"],
			Port:       port,
			AuthMethod: params["a"],
			Password:   params["p"],
			Key:        params["k"],
			Proxy:      params["proxy"],
		}
		// 检查
		serverConfig.Check()
		// 添加配置
		err := configs.AddServerConfig(alias, serverConfig)
		if err != nil {
			log.Fatalf("add ssh server config error: %s", err.Error())
		}
	} else {
		log.Fatalf("can't add ssh server config without arguments")
	}
}

// 删除配置
func doRemove(args []string) {
	if len(args) > 1 {
		alias := args[1]
		configs := ssh.LoadConfig()
		configs.RemoveServerConfig(alias)
	} else {
		log.Fatalf("can't remove ssh server config without arguments")
	}
}

func doConfig() {

}

// list servers
func doList(args []string) {
	configs := ssh.LoadConfig()
	servers := configs.Servers
	match := ""
	if len(args) > 1 {
		match = args[1]
	}
	for alias, server := range servers {
		if match == "" || strings.Contains(strings.ToLower(alias), strings.ToLower(match)) {
			info := fmt.Sprintf("[%s]%s@%s", alias, server.User, server.Host)
			if server.Port != 22 {
				info = info + "[:" + strconv.Itoa(server.Port) + "]"
			}
			if server.Password != "" {
				info = info + ":" + server.Password
			}
			if len(server.Proxy) > 0 {
				info = info + ", Proxy: " + server.Proxy
			}
			fmt.Println(info)
		}
	}
}

// help
func doHelp(args []string) {
	var helpUsage = fmt.Sprintf(usage, Version, BuildOn, User)
	if len(args) > 1 {
		switch args[1] {
		case "add":
			helpUsage = addUsage
		case "edit":
			helpUsage = editUsage
		}
	}
	fmt.Print(helpUsage)
}

// 透传
func passThrough(args []string) {
	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok {
		if status, ok := e.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		} else {
			os.Exit(1)
		}
	}
}
