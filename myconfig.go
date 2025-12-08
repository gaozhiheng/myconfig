package myconfig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gaozhiheng/vimcrypto"
)

// Config 配置管理器
type Config struct {
	configPath  string
	decryptPwd  string
	configData  map[string]interface{}
	keyFilePath string
}

var (
	globalConfig   *Config
	defaultKeyFile = "myconfigkey.json"
	// 编译时注入的密钥文件密码，通过 -ldflags 注入
	//go build -ldflags '-X github.com/gaozhiheng/myconfig.keyFilePassword=Gao@2025' example.go
	keyFilePassword = ""
)

// Init 初始化配置管理器
func Init(configPath, keyFilePath string) error {
	cfg := &Config{
		configPath: configPath,
	}

	// 如果未提供密钥文件路径，使用默认值
	if keyFilePath == "" {
		cfg.keyFilePath = defaultKeyFile
	} else {
		cfg.keyFilePath = keyFilePath
	}

	// 检查编译时是否注入了密钥文件密码
	if keyFilePassword == "" {
		return fmt.Errorf("key file password not set, please build with -ldflags '-X github.com/gaozhiheng/myconfig.keyFilePassword=your_password'")
	}

	// 检查密钥文件是否存在
	_, err := os.Stat(cfg.keyFilePath)
	if os.IsNotExist(err) {
		// 第一次初始化，要求用户输入解密密码
		const info = `
--------------------------------------------------------
企业级配置信息双重加密组件说明：
1.配置文件（一般为config.json）要求以blowfish2方式加密（文件头标识为“VimCrypt~03!”，使用vim工具:set cm=blowfish2，然后使用:X命令设置密码，即可加密保存文件）
  配置文件要求以json格式设置各配置项的key和value,如：
｛
	"APP-ID": "mwp2u2tgwmvjwle@#ESkwiw",
    "PORT": 30001
｝
2.本组件把配置文件的加密密码以不可告人的方式保存在密钥文件（一般为myconfigkey.json）
  如密钥文件缺失（当首次初始化或用户因变更了配置文件的密码而主动删除了密钥文件以便重新生成该文件时），本组件自动提示用户输入配置文件的密码。
--------------------------------------------------------
`
		fmt.Printf(info)
		fmt.Printf("密钥文件缺失，请提供配置文件 %s 所使用的加密密码: ", configPath)
		decryptPwd, err := readPasswordFromStdin()
		if err != nil {
			return fmt.Errorf("failed to read password: %v", err)
		}
		cfg.decryptPwd = decryptPwd

		// 创建并保存密钥文件
		if err := cfg.createKeyFile(); err != nil {
			return fmt.Errorf("failed to create key file: %v", err)
		}
		fmt.Printf("密钥文件 %s 创建成功\n", cfg.keyFilePath)
	} else if err != nil {
		return fmt.Errorf("failed to check key file: %v", err)
	} else {
		// 密钥文件存在，从中读取解密密码
		decryptPwd, err := cfg.readKeyFromFile()
		if err != nil {
			return fmt.Errorf("failed to read key from file: %v", err)
		}
		cfg.decryptPwd = decryptPwd
	}

	// 检查配置文件是否存在，如果不存在则创建空配置
	if err := cfg.ensureConfigFile(); err != nil {
		return fmt.Errorf("failed to ensure config file: %v", err)
	}

	// 加载配置数据
	if err := cfg.loadConfig(); err != nil {
		return err
	}

	globalConfig = cfg
	return nil
}

// ensureConfigFile 确保配置文件存在，如果不存在则创建空JSON对象
func (c *Config) ensureConfigFile() error {
	_, err := os.Stat(c.configPath)
	if os.IsNotExist(err) {
		fmt.Printf("配置文件 %s 不存在，创建空配置文件...\n", c.configPath)

		// 创建空JSON对象
		emptyConfig := make(map[string]interface{})
		configJSON, err := json.MarshalIndent(emptyConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal empty config: %v", err)
		}

		// 创建并加密配置文件
		file, err := os.Create(c.configPath)
		if err != nil {
			return fmt.Errorf("failed to create config file: %v", err)
		}
		defer file.Close()

		// 使用解密密码加密空配置
		err = vimcrypto.Encrypt(file, c.decryptPwd, configJSON)
		if err != nil {
			return fmt.Errorf("failed to encrypt config file: %v", err)
		}

		fmt.Printf("空配置文件 %s 创建成功\n", c.configPath)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to check config file: %v", err)
	}

	// 配置文件已存在，无需操作
	return nil
}

// createKeyFile 创建密钥文件
func (c *Config) createKeyFile() error {
	// 检查编译时是否注入了密钥文件密码
	if keyFilePassword == "" {
		return fmt.Errorf("key file password not set, please build with -ldflags '-X github.com/yourpackage/myconfig.keyFilePassword=your_password'")
	}

	file, err := os.Create(c.keyFilePath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}
	defer file.Close()

	// 使用编译时注入的密码加密解密密码
	err = vimcrypto.Encrypt(file, keyFilePassword, []byte(c.decryptPwd))
	if err != nil {
		return fmt.Errorf("failed to encrypt key file: %v", err)
	}

	return nil
}

// readKeyFromFile 从密钥文件读取解密密码
func (c *Config) readKeyFromFile() (string, error) {
	// 检查编译时是否注入了密钥文件密码
	if keyFilePassword == "" {
		return "", fmt.Errorf("key file password not set, please build with -ldflags '-X github.com/yourpackage/myconfig.keyFilePassword=your_password'")
	}

	file, err := os.Open(c.keyFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open key file: %v", err)
	}
	defer file.Close()

	keyData, err := vimcrypto.Decrypt(file, keyFilePassword, "utf-8")
	if err != nil {
		return "", fmt.Errorf("failed to decrypt key file: %v", err)
	}

	return strings.TrimSpace(keyData), nil
}

// readPasswordFromStdin 从标准输入读取密码（不显示）
func readPasswordFromStdin() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(password), nil
}

// SetPass 重置配置文件加密密码
func SetPass(newPassword string) error {
	if globalConfig == nil {
		return fmt.Errorf("config not initialized, call Init first")
	}

	// 检查编译时是否注入了密钥文件密码
	if keyFilePassword == "" {
		return fmt.Errorf("key file password not set")
	}

	// 将新密码加密保存到密钥文件
	file, err := os.Create(globalConfig.keyFilePath)
	if err != nil {
		return fmt.Errorf("failed to create key file: %v", err)
	}
	defer file.Close()

	// 使用编译时注入的密码加密新密码
	err = vimcrypto.Encrypt(file, keyFilePassword, []byte(newPassword))
	if err != nil {
		return fmt.Errorf("failed to encrypt new password: %v", err)
	}

	// 更新内存中的解密密码
	globalConfig.decryptPwd = newPassword

	// 重新加载配置，因为解密密码改变了
	return globalConfig.loadConfig()
}

// Get 获取配置值
func Get(key string) interface{} {
	if globalConfig == nil {
		log.Fatal("config not initialized, call Init first")
	}

	value, exists := globalConfig.configData[key]
	if !exists {
		log.Fatalf("config key '%s' not found", key)
	}
	return value
}

// GetString 获取字符串类型配置值
func GetString(key string) string {
	value := Get(key)

	str, ok := value.(string)
	if !ok {
		log.Fatalf("config key '%s' is not string, got %T", key, value)
	}
	return str
}

// GetInt 获取整数类型配置值
func GetInt(key string) int {
	value := Get(key)

	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	default:
		log.Fatalf("config key '%s' is not integer, got %T", key, value)
		return 0
	}
}

// GetBool 获取布尔类型配置值
func GetBool(key string) bool {
	value := Get(key)

	b, ok := value.(bool)
	if !ok {
		log.Fatalf("config key '%s' is not boolean, got %T", key, value)
	}
	return b
}

// GetMap 获取map类型配置值
func GetMap(key string) map[string]interface{} {
	value := Get(key)

	m, ok := value.(map[string]interface{})
	if !ok {
		log.Fatalf("config key '%s' is not map, got %T", key, value)
	}
	return m
}

// GetArray 获取数组类型配置值
func GetArray(key string) []interface{} {
	value := Get(key)

	arr, ok := value.([]interface{})
	if !ok {
		log.Fatalf("config key '%s' is not array, got %T", key, value)
	}
	return arr
}

// SetConfig 设置配置项
func SetConfig(key string, value interface{}) error {
	if globalConfig == nil {
		return fmt.Errorf("config not initialized, call Init first")
	}

	globalConfig.configData[key] = value
	return globalConfig.saveConfig()
}

// DelConfig 删除配置项
func DelConfig(key string) error {
	if globalConfig == nil {
		return fmt.Errorf("config not initialized, call Init first")
	}

	delete(globalConfig.configData, key)
	return globalConfig.saveConfig()
}

// loadConfig 加载配置数据
func (c *Config) loadConfig() error {
	file, err := os.Open(c.configPath)
	if err != nil {
		return fmt.Errorf("failed to open config file: %v", err)
	}
	defer file.Close()

	// 解密文件内容
	textData, err := vimcrypto.Decrypt(file, c.decryptPwd, "utf-8")
	if err != nil {
		return fmt.Errorf("failed to decrypt config file: %v", err)
	}

	c.configData = make(map[string]interface{})
	if err := json.Unmarshal([]byte(textData), &c.configData); err != nil {
		return fmt.Errorf("invalid JSON format in config file: %v", err)
	}

	return nil
}

// saveConfig 保存配置到文件
func (c *Config) saveConfig() error {
	// 序列化配置数据
	configJSON, err := json.MarshalIndent(c.configData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config data: %v", err)
	}

	// 创建配置文件
	file, err := os.Create(c.configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %v", err)
	}
	defer file.Close()

	// 加密并写入配置
	err = vimcrypto.Encrypt(file, c.decryptPwd, configJSON)
	if err != nil {
		return fmt.Errorf("failed to encrypt config file: %v", err)
	}

	return nil
}

// GetConfigData 获取原始配置数据（用于兼容旧代码）
func GetConfigData() map[string]interface{} {
	if globalConfig == nil {
		log.Fatal("config not initialized, call Init first")
	}
	return globalConfig.configData
}
