package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

// Config 定义配置结构
type Config struct {
	OpenApi struct {
		ApiKey  string `yaml:"api_key"`
		BaseUrl string `yaml:"base_url"`
	} `yaml:"open_api"`
}

// LoadConfig 根据环境加载配置文件
func LoadConfig() (*Config, error) {
	// 获取环境变量 ENV
	env := os.Getenv("ENV")
	// print env
	fmt.Println("env:", env)
	// 默认配置文件路径
	var configFile string
	if env == "" {
		// 如果未设置 ENV，加载默认的 config.yml
		configFile = filepath.Join("conf", "config.yml")
	} else {
		// 加载对应环境的配置文件
		configFile = filepath.Join("conf", fmt.Sprintf("config-%s.yml", env))
	}

	// 读取文件内容
	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// 解析 YAML
	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	return &config, nil
}
