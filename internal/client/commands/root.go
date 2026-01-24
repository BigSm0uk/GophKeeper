package commands

import (
	"fmt"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	clientapp "github.com/BigSm0uk/GophKeeper/internal/client/app"
	"github.com/BigSm0uk/GophKeeper/internal/client/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/client/app/logger"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/spf13/cobra"
)

var (
	rootCmd   = &cobra.Command{Use: "gophkeeper", Short: "GophKeeper client"}
	cfgPath   string
	server    string
	insecure  bool
	logLevel  string
	buildInfo = struct {
		version   string
		buildDate string
		commit    string
	}{}

	container *clientapp.Container
)

// SetBuildInfo прокидывает данные сборки из main.
func SetBuildInfo(version, date, commit string) {
	buildInfo.version = version
	buildInfo.buildDate = date
	buildInfo.commit = commit
}

// Execute запускает корневую команду.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to client config (optional)")
	rootCmd.PersistentFlags().StringVar(&server, "server", "", "server address host:port (override config)")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "disable TLS for gRPC")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level (debug|info|warn|error)")

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Инициализируем только один раз.
		if container != nil {
			return nil
		}

		cfg, err := config.Load(cfgPath)
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		if server != "" {
			cfg.ServerAddress = server
		}
		if cmd.Flags().Changed("insecure") {
			cfg.Insecure = insecure
		}
		if logLevel != "" {
			cfg.Logger.Level = logLevel
		}

		log, err := logger.NewZapLogger(cfg.Logger.Level, cfg.Env == config.EnvDevelopment)
		if err != nil {
			return fmt.Errorf("logger: %w", err)
		}

		client, err := api.New(cfg.ServerAddress, cfg.Insecure, cfg.RequestTimeout, log)
		if err != nil {
			return fmt.Errorf("api client: %w", err)
		}

		tokenStore, err := storage.NewTokenStore()
		if err != nil {
			return fmt.Errorf("token store: %w", err)
		}

		container = clientapp.NewContainer(log, cfg, client, tokenStore)
		return nil
	}

	// Команда version
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print client version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("GophKeeper client\nVersion: %s\nBuildDate: %s\nCommit: %s\n",
				buildInfo.version, buildInfo.buildDate, buildInfo.commit)
		},
	})

	// Auth команды
	rootCmd.AddCommand(authCmd)
	// Credentials команды
	rootCmd.AddCommand(credCmd)
	// Cards команды
	rootCmd.AddCommand(cardCmd)
	// Text команды
	rootCmd.AddCommand(textCmd)
}
