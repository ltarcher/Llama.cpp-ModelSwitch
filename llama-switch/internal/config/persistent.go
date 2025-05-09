package config

import (
	"encoding/json"
	"fmt"
	"llama-switch/internal/model"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// 配置文件版本
	ConfigVersion = "1.0.0"
	// 配置文件名
	ConfigFileName = "model_persistent.json"
	// 备份文件后缀
	BackupSuffix = ".backup"
)

// PersistentModelConfig 持久化配置结构
type PersistentModelConfig struct {
	Version    string                     `json:"version"`     // 配置版本号
	UpdateTime string                     `json:"update_time"` // 最后更新时间
	Models     map[string]ModelConfigItem `json:"models"`      // 模型配置映射
}

// ModelConfigItem 模型配置项
type ModelConfigItem struct {
	ModelConfig *model.ModelConfig `json:"model_config"` // 完整的模型配置
	LastStatus  model.ModelStatus  `json:"last_status"`  // 最后运行状态
}

// PersistentManager 持久化管理器
type PersistentManager struct {
	config *Config
	mu     sync.RWMutex
}

// NewPersistentManager 创建新的持久化管理器
func NewPersistentManager(cfg *Config) *PersistentManager {
	return &PersistentManager{
		config: cfg,
	}
}

// LoadConfig 加载配置
func (pm *PersistentManager) LoadConfig() (*PersistentModelConfig, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// 确保配置目录存在
	configDir := filepath.Join(filepath.Dir(pm.config.ModelsDir), "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	// 检查配置文件是否存在
	configPath := filepath.Join(configDir, ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 如果配置文件不存在，创建新的配置
		return &PersistentModelConfig{
			Version:    ConfigVersion,
			UpdateTime: time.Now().Format(time.RFC3339),
			Models:     make(map[string]ModelConfigItem),
		}, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(filepath.Join(configDir, ConfigFileName))
	if err != nil {
		// 尝试从备份文件恢复
		backupPath := configPath + BackupSuffix
		if backupData, backupErr := os.ReadFile(backupPath); backupErr == nil {
			data = backupData
		} else {
			return nil, fmt.Errorf("failed to read config file and backup: %v", err)
		}
	}

	// 解析配置
	var config PersistentModelConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	// 版本兼容性检查
	if config.Version != ConfigVersion {
		// TODO: 实现版本迁移逻辑
		return nil, fmt.Errorf("unsupported config version: %s", config.Version)
	}

	return &config, nil
}

// SaveConfig 保存配置
func (pm *PersistentManager) SaveConfig(config *PersistentModelConfig) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// 更新时间戳
	config.UpdateTime = time.Now().Format(time.RFC3339)

	// 序列化配置
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	// 确保配置目录存在并设置配置路径
	configDir := filepath.Join(filepath.Dir(pm.config.ModelsDir), "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, ConfigFileName)

	// 如果存在旧配置，先备份
	if _, err := os.Stat(configPath); err == nil {
		if err := os.Rename(configPath, configPath+BackupSuffix); err != nil {
			return fmt.Errorf("failed to backup old config: %v", err)
		}
	}

	// 写入新配置
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// UpdateModelConfig 更新模型配置
func (pm *PersistentManager) UpdateModelConfig(modelName string, config *model.ModelConfig, status *model.ModelStatus) error {
	persistentConfig, err := pm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config for update: %v", err)
	}

	if status == nil {
		return fmt.Errorf("model status cannot be nil")
	}

	persistentConfig.Models[modelName] = ModelConfigItem{
		ModelConfig: config,
		LastStatus:  *status,
	}

	return pm.SaveConfig(persistentConfig)
}

// RemoveModelConfig 移除模型配置
func (pm *PersistentManager) RemoveModelConfig(modelName string) error {
	persistentConfig, err := pm.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config for removal: %v", err)
	}

	delete(persistentConfig.Models, modelName)
	return pm.SaveConfig(persistentConfig)
}

// GetModelConfigs 获取所有模型配置
func (pm *PersistentManager) GetModelConfigs() (map[string]ModelConfigItem, error) {
	persistentConfig, err := pm.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load model configs: %v", err)
	}

	return persistentConfig.Models, nil
}
