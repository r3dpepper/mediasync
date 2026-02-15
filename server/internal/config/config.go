package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Discovery DiscoveryConfig `mapstructure:"discovery"`
	Upload    UploadConfig    `mapstructure:"upload"`
	Thumbnail ThumbnailConfig `mapstructure:"thumbnail"`
	Backup    BackupConfig    `mapstructure:"backup"`
	Security  SecurityConfig  `mapstructure:"security"`
}

type ServerConfig struct {
	Port        int    `mapstructure:"port"`
	BindAddress string `mapstructure:"bind_address"`
	LogLevel    string `mapstructure:"log_level"`
}

type StorageConfig struct {
	PrimaryPath string `mapstructure:"primary_path"`
	BackupPath  string `mapstructure:"backup_path"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type DiscoveryConfig struct {
	ServiceName string `mapstructure:"service_name"`
	Enabled     bool   `mapstructure:"enabled"`
}

type UploadConfig struct {
	MaxSizeMB          int  `mapstructure:"max_size_mb"`
	MaxConcurrent      int  `mapstructure:"max_concurrent"`
	TimeoutSeconds     int  `mapstructure:"timeout_seconds"`
	VerifyDuringUpload bool `mapstructure:"verify_during_upload"`
}

type ThumbnailConfig struct {
	MaxWorkers int    `mapstructure:"max_workers"`
	Quality    string `mapstructure:"quality"`
	CachePath  string `mapstructure:"cache_path"`
}

type BackupConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	ScheduleCron    string `mapstructure:"schedule_cron"`
	VerifyAfterCopy bool   `mapstructure:"verify_after_copy"`
}

type SecurityConfig struct {
	AuthEnabled  bool           `mapstructure:"auth_enabled"`
	AuthType     string         `mapstructure:"auth_type"`
	AllowedHosts []string       `mapstructure:"allowed_hosts"`
	Users        []SecurityUser `mapstructure:"users"`
}

type SecurityUser struct {
	Username     string `mapstructure:"username"`
	PasswordHash string `mapstructure:"password_hash"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".media-server")

	return &Config{
		Server: ServerConfig{
			Port:        8080,
			BindAddress: "0.0.0.0",
			LogLevel:    "info",
		},
		Storage: StorageConfig{
			PrimaryPath: "",
			BackupPath:  "",
		},
		Database: DatabaseConfig{
			Path: filepath.Join(configDir, "media.db"),
		},
		Discovery: DiscoveryConfig{
			ServiceName: "media-server",
			Enabled:     true,
		},
		Upload: UploadConfig{
			MaxSizeMB:          5000,
			MaxConcurrent:      5,
			TimeoutSeconds:     600,
			VerifyDuringUpload: true,
		},
		Thumbnail: ThumbnailConfig{
			MaxWorkers: 4,
			Quality:    "high",
			CachePath:  filepath.Join(configDir, "thumbnails"),
		},
		Backup: BackupConfig{
			Enabled:         false,
			ScheduleCron:    "0 2 * * *", // 2 AM daily
			VerifyAfterCopy: true,
		},
		Security: SecurityConfig{
			AuthEnabled:  false,
			AuthType:     "basic",
			AllowedHosts: []string{},
			Users:        []SecurityUser{},
		},
	}
}

func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".media-server", "config.yaml")
}

func LoadConfig(cfgFile string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		configDir := filepath.Join(homeDir, ".media-server")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func SaveConfig(cfg *Config) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(homeDir, ".media-server")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.yaml")
	
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	v.Set("server.port", cfg.Server.Port)
	v.Set("server.bind_address", cfg.Server.BindAddress)
	v.Set("server.log_level", cfg.Server.LogLevel)
	v.Set("storage.primary_path", cfg.Storage.PrimaryPath)
	v.Set("storage.backup_path", cfg.Storage.BackupPath)
	v.Set("database.path", cfg.Database.Path)
	v.Set("discovery.service_name", cfg.Discovery.ServiceName)
	v.Set("discovery.enabled", cfg.Discovery.Enabled)
	v.Set("upload.max_size_mb", cfg.Upload.MaxSizeMB)
	v.Set("upload.max_concurrent", cfg.Upload.MaxConcurrent)
	v.Set("upload.timeout_seconds", cfg.Upload.TimeoutSeconds)
	v.Set("upload.verify_during_upload", cfg.Upload.VerifyDuringUpload)
	v.Set("thumbnail.max_workers", cfg.Thumbnail.MaxWorkers)
	v.Set("thumbnail.quality", cfg.Thumbnail.Quality)
	v.Set("thumbnail.cache_path", cfg.Thumbnail.CachePath)
	v.Set("backup.enabled", cfg.Backup.Enabled)
	v.Set("backup.schedule_cron", cfg.Backup.ScheduleCron)
	v.Set("backup.verify_after_copy", cfg.Backup.VerifyAfterCopy)
	v.Set("security.auth_enabled", cfg.Security.AuthEnabled)
	v.Set("security.auth_type", cfg.Security.AuthType)
	v.Set("security.allowed_hosts", cfg.Security.AllowedHosts)
	v.Set("security.users", cfg.Security.Users)

	return v.WriteConfigAs(configPath)
}

func validateConfig(cfg *Config) error {
	if cfg.Storage.PrimaryPath == "" {
		return fmt.Errorf("storage.primary_path cannot be empty")
	}

	if _, err := os.Stat(cfg.Storage.PrimaryPath); os.IsNotExist(err) {
		return fmt.Errorf("storage.primary_path does not exist: %s", cfg.Storage.PrimaryPath)
	}

	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	// Create thumbnail cache directory if it doesn't exist
	if err := os.MkdirAll(cfg.Thumbnail.CachePath, 0755); err != nil {
		return fmt.Errorf("failed to create thumbnail cache directory: %w", err)
	}

	return nil
}
