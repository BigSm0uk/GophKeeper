package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

func init() {
	var tuiRegister bool
	var tuiLogin bool
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Open interactive TUI for login/registration",
		Long:  "Opens an interactive terminal UI where you can choose to login or register",
		RunE: func(cmd *cobra.Command, args []string) error {
			if container == nil || container.TokenStore == nil {
				return fmt.Errorf("client not initialized")
			}
			store := container.TokenStore

			// Check for cached session — skip auth TUI if user has logged in before
			if !tuiRegister && !tuiLogin {
				username, offErr := tryOfflineEntry()
				if offErr == nil && username != "" {
					if container.Logger != nil {
						container.Logger.Info("cached session found, skipping auth", zap.String("username", username))
					}
					return runMasterPasswordAndMainMenu(username)
				}
			}

			mode := tui.ModeSelect
			if tuiRegister {
				mode = tui.ModeRegister
			} else if tuiLogin {
				mode = tui.ModeLogin
			}

			model := tui.NewAuthModel(container.API, store, mode)
			p := tea.NewProgram(model, tea.WithAltScreen())
			finalModel, err := p.Run()
			if err != nil {
				if container.Logger != nil {
					container.Logger.Debug("auth TUI program finished with error", zap.Error(err))
				}
				return fmt.Errorf("tui: %w", err)
			}

			if container.Logger != nil {
				container.Logger.Debug("auth TUI program finished successfully")
			}

			if authResult, ok := tui.ExtractAuthResult(finalModel); ok && authResult.Success {
				container.API.SetAccessToken(authResult.AccessToken)
				if container.Logger != nil {
					container.Logger.Info("authentication successful", zap.String("username", authResult.Username))
				}

				return runMasterPasswordAndMainMenu(authResult.Username)
			}

			// Fallback: read tokens from keyring with retries (macOS keyring delay)
			var username string
			var token string
			var errFetch error

			for i := 0; i < 5; i++ {
				time.Sleep(300 * time.Millisecond)

				username, errFetch = store.GetCurrentUsername()
				if errFetch == nil && username != "" {
					token, errFetch = store.GetAccessToken(username)
					if errFetch == nil && token != "" {
						break
					}
				}

				if container.Logger != nil {
					container.Logger.Debug("retrying to get auth data",
						zap.Int("attempt", i+1),
						zap.String("username", username),
						zap.Error(errFetch),
					)
				}
			}

			if errFetch != nil || username == "" || token == "" {
				if container.Logger != nil {
					container.Logger.Debug("auth data not found, exiting",
						zap.String("username", username),
						zap.Bool("has_token", token != ""),
						zap.Error(errFetch),
					)
				}
				return nil
			}

			container.API.SetAccessToken(token)
			if container.Logger != nil {
				container.Logger.Info("authentication successful", zap.String("username", username))
			}

			return runMasterPasswordAndMainMenu(username)
		},
	}
	tuiCmd.Flags().BoolVar(&tuiRegister, "register", false, "open registration mode directly")
	tuiCmd.Flags().BoolVar(&tuiLogin, "login", false, "open login mode directly")

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, err := prompt("Username")
			if err != nil {
				return err
			}
			password, err := promptSecret("Password")
			if err != nil {
				return err
			}
			email, _ := prompt("Email (optional)")

			resp, err := container.API.Register(context.Background(), username, password, email)
			if err != nil {
				return fmt.Errorf("register: %w", err)
			}

			container.Logger.Info("user registered",
				zap.String("user_id", resp.UserId),
				zap.String("username", resp.Username),
			)
			fmt.Println("✅ Registered")
			return nil
		},
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Login and store tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, err := prompt("Username")
			if err != nil {
				return err
			}
			password, err := promptSecret("Password")
			if err != nil {
				return err
			}

			tokens, err := container.API.PasswordToken(context.Background(), username, password, "gophkeeper-cli", "", "offline_access")
			if err != nil {
				return fmt.Errorf("login: %w", err)
			}

			if container.TokenStore == nil {
				return fmt.Errorf("token store not initialized")
			}
			if err := container.TokenStore.SaveAccessToken(username, tokens.AccessToken); err != nil {
				return fmt.Errorf("save access token: %w", err)
			}

			container.API.SetAccessToken(tokens.AccessToken)
			container.Logger.Info("user logged in", zap.String("username", username))
			fmt.Println("✅ Logged in")
			return nil
		},
	}

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout and revoke refresh token",
		RunE: func(cmd *cobra.Command, args []string) error {
			username, err := prompt("Username for logout")
			if err != nil {
				return err
			}

			if container.TokenStore == nil {
				return fmt.Errorf("token store not initialized")
			}

			container.TokenStore.DeleteTokens(username)
			container.Logger.Info("user logged out", zap.String("username", username))
			fmt.Println("✅ Logged out")
			return nil
		},
	}

	authCmd.AddCommand(registerCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(tuiCmd)
}

func prompt(label string) (string, error) {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

// promptSecret без сокрытия ввода (минимум зависимостей; позже можно заменить).
func promptSecret(label string) (string, error) {
	fmt.Printf("%s: ", label)
	bytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

// promptRequired prompts for required input and validates it's not empty
func promptRequired(label string) string {
	for {
		value, err := prompt(label)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		if value == "" {
			fmt.Printf("%s is required\n", label)
			continue
		}
		return value
	}
}
