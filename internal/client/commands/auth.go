package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/BigSm0uk/GophKeeper/internal/client/storage"
	"github.com/BigSm0uk/GophKeeper/internal/client/tui"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/term"
)

var (
	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}
)

func init() {
	var tuiRegister bool
	tuiCmd := &cobra.Command{
		Use:   "tui",
		Short: "Open TUI for login/registration",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storage.NewTokenStore()
			if err != nil {
				return fmt.Errorf("keyring: %w", err)
			}
			mode := tui.ModeLogin
			if tuiRegister {
				mode = tui.ModeRegister
			}
			model := tui.NewAuthModel(container.API, store, mode)
			if _, err := tea.NewProgram(model).Run(); err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			return nil
		},
	}
	tuiCmd.Flags().BoolVar(&tuiRegister, "register", false, "open registration mode")

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

			store, err := storage.NewTokenStore()
			if err != nil {
				return fmt.Errorf("keyring: %w", err)
			}
			if err := store.SaveAccessToken(username, tokens.AccessToken); err != nil {
				return fmt.Errorf("save access token: %w", err)
			}
			if err := store.SaveRefreshToken(username, tokens.RefreshToken); err != nil {
				return fmt.Errorf("save refresh token: %w", err)
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

			store, err := storage.NewTokenStore()
			if err != nil {
				return fmt.Errorf("keyring: %w", err)
			}
			refresh, err := store.GetRefreshToken(username)
			if err != nil {
				return fmt.Errorf("get refresh token: %w", err)
			}

			if _, err := container.API.Revoke(context.Background(), refresh, "gophkeeper-cli", "", pb.TokenTypeHint_TOKEN_TYPE_HINT_REFRESH_TOKEN); err != nil {
				return fmt.Errorf("revoke: %w", err)
			}
			store.DeleteTokens(username)
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
