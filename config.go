package ssh

import (
	"github.com/tj/go-config"
	"go-ssh/utils"
	"log"
)

// Default Config File
const ConfigFile = "~/.ssh/ssh_config.json"

// 服务器配置
type ServerConfig struct {
	// 服务器IP
	Host string `json:"host"`
	// 用户名
	User string `json:"user"`
	// 端口
	Port int `json:"port"`
	// 授权方式
	AuthMethod string `json:"auth_method"`
	// 密码
	Password string `json:"password"`
	// 秘钥
	Key string `json:"key"`
	// 代理
	Proxy string `json:"proxy"`
}

// 全局配置
type GlobalConfig struct {
	// 定时心跳
	ServerAliveInterval int `json:"server_alive_interval"`
}

// 配置
type Configs struct {
	// 全局配置
	Global *GlobalConfig `json:"global"`
	// 服务器配置
	Servers map[string]*ServerConfig `json:"servers"`
}

// 加载配置
func LoadConfig() *Configs {
	configs := &Configs{
		Servers: make(map[string]*ServerConfig, 0),
	}
	path, err := utils.ParseAbsolutePath(ConfigFile)
	if err != nil {
		log.Fatalf("find path error %s", err.Error())
	}
	if !utils.Exists(path) {
		return configs
	}
	if utils.IsDir(path) {
		log.Fatalf("path is a dir: %s", path)
	}
	err = config.Load(path, configs)
	if err != nil {
		log.Fatalf("load configs error %s", err.Error())
	}
	return configs
}

// 保存配置
func (configs *Configs) SaveConfig() error {
	// 初始化数据
	if configs.Global == nil {
		configs.Global = &GlobalConfig{
			ServerAliveInterval: 30,
		}
	}
	path, err := utils.ParseAbsolutePath(ConfigFile)
	if err != nil {
		log.Fatalf("find path error %s", err.Error())
	}
	return config.Save(path, configs)
}

// 增加配置
func (configs *Configs) AddServerConfig(alias string, serverConfig *ServerConfig) error {
	if serverConfig == nil {
		 return nil
	}
	// 初始化端口
	if serverConfig.Port < 1 {
		serverConfig.Port = 22
	}
	configs.Servers[alias] = serverConfig
	return configs.SaveConfig()
}

// 删除服务器配置
func (configs *Configs) RemoveServerConfig(alias string) error {
	if len(alias) == 0 {
		return nil
	}
	delete(configs.Servers, alias)
	return configs.SaveConfig()
}

// 参数校验
func (serverConfig *ServerConfig) Check() {
	if len(serverConfig.Host) == 0 {
		log.Fatalf("ssh server host is not empty")
	}
	if len(serverConfig.User) == 0 {
		log.Fatalf("ssh server user is not empty")
	}
	// no password
	if len(serverConfig.Password) == 0 {
		if len(serverConfig.AuthMethod) == 0 {
			serverConfig.AuthMethod = KeyAuth
		} else if serverConfig.AuthMethod == PasswordAuth {
			log.Fatalf("ssh server password is not empty")
		}
	} else if len(serverConfig.AuthMethod) == 0 {
		serverConfig.AuthMethod = PasswordAuth
	}
	if serverConfig.AuthMethod != KeyAuth && serverConfig.AuthMethod != PasswordAuth {
		log.Fatalf("ssh server authmethod error: %s", serverConfig.AuthMethod)
	}
}
