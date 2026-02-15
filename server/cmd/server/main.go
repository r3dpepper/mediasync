package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yourusername/private-media-ecosystem/internal/api"
	"github.com/yourusername/private-media-ecosystem/internal/config"
	"github.com/yourusername/private-media-ecosystem/internal/db"
	"github.com/yourusername/private-media-ecosystem/internal/discovery"
	"github.com/yourusername/private-media-ecosystem/internal/worker"
)

var (
	version = "1.0.0"
	cfgFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "media-server",
		Short:   "Private Media Ecosystem Server",
		Version: version,
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.media-server/config.yaml)")

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(configCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			storage, _ := cmd.Flags().GetString("storage")
			backup, _ := cmd.Flags().GetString("backup")
			port, _ := cmd.Flags().GetInt("port")
			mdnsName, _ := cmd.Flags().GetString("mdns-name")

			cfg := config.DefaultConfig()
			cfg.Storage.PrimaryPath = storage
			cfg.Storage.BackupPath = backup
			cfg.Server.Port = port
			cfg.Discovery.ServiceName = mdnsName

			if err := config.SaveConfig(cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			log.Info().Msg("Configuration initialized successfully")
			log.Info().Str("config_path", config.GetConfigPath()).Msg("Config saved to")

			// Initialize database
			database, err := db.InitDB(cfg.Database.Path)
			if err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}
			defer db.Close(database)

			log.Info().Msg("Database initialized successfully")
			return nil
		},
	}

	cmd.Flags().String("storage", "", "Path to primary storage (My Passport)")
	cmd.Flags().String("backup", "", "Path to backup storage (optional)")
	cmd.Flags().Int("port", 8080, "HTTP server port")
	cmd.Flags().String("mdns-name", "media-server", "mDNS service name")
	cmd.MarkFlagRequired("storage")

	return cmd
}

func startCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the media server",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Setup logging
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

			// Load configuration
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			log.Info().Str("version", version).Msg("Starting Private Media Ecosystem Server")
			log.Info().Str("primary_path", cfg.Storage.PrimaryPath).Msg("Storage configuration")
			log.Info().Str("backup_path", cfg.Storage.BackupPath).Msg("Backup configuration")
			log.Info().Int("port", cfg.Server.Port).Msg("Server configuration")
			log.Info().Str("db_path", cfg.Database.Path).Msg("Database configuration")

			// Initialize database
			database, err := db.InitDB(cfg.Database.Path)
			if err != nil {
				return fmt.Errorf("failed to initialize database: %w", err)
			}
			defer db.Close(database)

			// Start mDNS service
			mdnsServer, err := discovery.StartMDNS(cfg.Discovery.ServiceName, cfg.Server.Port)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to start mDNS service (discovery will not work)")
			} else {
				defer mdnsServer.Shutdown()
				log.Info().Str("service", cfg.Discovery.ServiceName+".local").Msg("mDNS service started")
			}

			// Start background workers
			workerManager := worker.NewManager(cfg, database)
			workerManager.Start()
			defer workerManager.Stop()

			// Start HTTP server
			server := api.NewServer(cfg, database)

			// Graceful shutdown
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			go func() {
				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
				<-sigChan
				log.Info().Msg("Shutdown signal received")
				cancel()
			}()

			return server.Start(ctx)
		},
	}

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage server configuration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig(cfgFile)
			if err != nil {
				return err
			}
			fmt.Printf("Configuration:\n%+v\n", cfg)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			viper.Set(key, value)
			return viper.WriteConfig()
		},
	})

	return cmd
}
