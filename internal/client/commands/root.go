package commands

import (
	"fmt"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/api"
	clientapp "github.com/BigSm0uk/GophKeeper/internal/client/app"
	"github.com/BigSm0uk/GophKeeper/internal/client/app/config"
	"github.com/BigSm0uk/GophKeeper/internal/client/app/logger"
	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/BigSm0uk/GophKeeper/internal/client/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "gophkeeper",
		Short: "GophKeeper client",
		Long:  "GophKeeper - secure password manager client",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Если нет подкоманд, запускаем интерактивный UI
			if len(args) == 0 {
				if container == nil || container.TokenStore == nil {
					return fmt.Errorf("client not initialized. Please run: gophkeeper auth tui")
				}
				model := tui.NewAuthModel(container.API, container.TokenStore, tui.ModeSelect)
				finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
				if err != nil {
					return fmt.Errorf("tui: %w", err)
				}

				if authResult, ok := tui.ExtractAuthResult(finalModel); ok && authResult.Success {
					container.API.SetAccessToken(authResult.AccessToken)

					// Запрашиваем мастер-пароль для инициализации локального хранилища
					dbPath, err := container.Config.GetLocalDBPath()
					if err != nil {
						return fmt.Errorf("failed to get db path: %w", err)
					}

					masterPassModel := tui.NewMasterPasswordModel(authResult.Username, dbPath, container.TokenStore)
					mpFinal, err := tea.NewProgram(masterPassModel, tea.WithAltScreen()).Run()
					if err != nil {
						return fmt.Errorf("master password: %w", err)
					}

					// Проверяем успешность ввода мастер-пароля
					if mpModel, ok := mpFinal.(tui.MasterPasswordModel); ok && mpModel.IsSuccess() {
						storageManager := mpModel.GetStorageManager()
						if storageManager != nil {
							container.StorageManager = storageManager
						}
					} else {
						return fmt.Errorf("master password entry failed or cancelled")
					}

					offlineService := container.GetOfflineService()
					mainMenu := tui.NewMainMenuModel(container.API, container.TokenStore, offlineService, container.StorageManager, container.SyncManager)
					if _, err := tea.NewProgram(mainMenu, tea.WithAltScreen()).Run(); err != nil {
						return fmt.Errorf("main menu: %w", err)
					}
					
					// Очищаем ресурсы после выхода
					if err := container.Close(); err != nil {
						return fmt.Errorf("failed to close container: %w", err)
					}
					
					return nil
				}

				// Fallback: читаем токены из keyring с ретраями
				var username string
				var token string
				var errFetch error
				for i := 0; i < 5; i++ {
					time.Sleep(300 * time.Millisecond)
					username, errFetch = container.TokenStore.GetCurrentUsername()
					if errFetch == nil && username != "" {
						token, errFetch = container.TokenStore.GetAccessToken(username)
						if errFetch == nil && token != "" {
							break
						}
					}
				}
				if errFetch != nil || username == "" || token == "" {
					return nil
				}
				container.API.SetAccessToken(token)

				// Запрашиваем мастер-пароль для разблокировки хранилища
				dbPath, err := container.Config.GetLocalDBPath()
				if err != nil {
					return fmt.Errorf("failed to get db path: %w", err)
				}

				masterPassModel := tui.NewMasterPasswordModel(username, dbPath, container.TokenStore)
				mpFinal, err := tea.NewProgram(masterPassModel, tea.WithAltScreen()).Run()
				if err != nil {
					return fmt.Errorf("master password: %w", err)
				}

				// Проверяем успешность ввода мастер-пароля
				if mpModel, ok := mpFinal.(tui.MasterPasswordModel); ok && mpModel.IsSuccess() {
					storageManager := mpModel.GetStorageManager()
					if storageManager != nil {
						container.StorageManager = storageManager
					}
				} else {
					return fmt.Errorf("master password entry failed or cancelled")
				}

				offlineService := container.GetOfflineService()
				mainMenu := tui.NewMainMenuModel(container.API, container.TokenStore, offlineService, container.StorageManager, container.SyncManager)
				if _, err := tea.NewProgram(mainMenu, tea.WithAltScreen()).Run(); err != nil {
					return fmt.Errorf("main menu: %w", err)
				}
				
				// Очищаем ресурсы после выхода
				if err := container.Close(); err != nil {
					return fmt.Errorf("failed to close container: %w", err)
				}
				
				return nil
			}
			return cmd.Help()
		},
	}
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
