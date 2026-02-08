package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/spf13/cobra"
)

var credCmd = &cobra.Command{
	Use:   "cred",
	Short: "Manage credentials (login/password pairs)",
}

var credAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new credential",
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		// Инициализируем encryptor
		if err := ensureEncryptor(); err != nil {
			return err
		}

		name := promptRequired("Name")
		login := promptRequired("Login")
		password, err := promptSecret("Password")
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		if password == "" {
			return fmt.Errorf("password is required")
		}
		url, _ := cmd.Flags().GetString("url")
		metadata, _ := cmd.Flags().GetString("metadata")

		// Шифруем чувствительные данные
		encryptedPassword, err := container.Encryptor.Encrypt(password)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}

		req := &pb.CredentialCreateRequest{
			Name:     name,
			Login:    login,
			Password: encryptedPassword, // отправляем зашифрованный пароль
		}
		if url != "" {
			req.Url = &url
		}
		if metadata != "" {
			req.Metadata = &metadata
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := container.API.CreateCredential(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to create credential: %w", err)
		}

		fmt.Printf("✓ Credential created successfully!\n")
		fmt.Printf("  GetID:    %s\n", resp.Credential.Id)
		fmt.Printf("  Name:  %s\n", resp.Credential.Name)
		fmt.Printf("  Login: %s\n", resp.Credential.Login)

		return nil
	},
}

var credListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		limit, _ := cmd.Flags().GetUint32("limit")
		offset, _ := cmd.Flags().GetUint32("offset")

		if limit == 0 {
			limit = 20
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := container.API.ListCredentials(ctx, limit, offset)
		if err != nil {
			return fmt.Errorf("failed to list credentials: %w", err)
		}

		if len(resp.Items) == 0 {
			fmt.Println("No credentials found.")
			return nil
		}

		fmt.Printf("Total credentials: %d\n\n", resp.Page.Total)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GetID\tNAME\tLOGIN\tURL\tUPDATED")
		fmt.Fprintln(w, "--\t----\t-----\t---\t-------")

		for _, item := range resp.Items {
			updatedAt := "N/A"
			if item.UpdatedAt != nil {
				updatedAt = item.UpdatedAt.AsTime().Format("2006-01-02 15:04")
			}

			idShort := item.Id
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}

			url := ""
			if item.Url != nil {
				url = *item.Url
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				idShort, item.Name, item.Login, url, updatedAt)
		}
		w.Flush()

		if offset+limit < resp.Page.Total {
			fmt.Printf("\nShowing %d-%d of %d. Use --offset to see more.\n", offset+1, offset+uint32(len(resp.Items)), resp.Page.Total)
		}

		return nil
	},
}

var credGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a credential by GetID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		// Инициализируем encryptor
		if err := ensureEncryptor(); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := container.API.GetCredential(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get credential: %w", err)
		}

		cred := resp.Credential

		// Дешифруем пароль
		decryptedPassword, err := container.Encryptor.Decrypt(cred.Password)
		if err != nil {
			return fmt.Errorf("failed to decrypt password: %w", err)
		}

		fmt.Printf("Credential Details:\n")
		fmt.Printf("  GetID:       %s\n", cred.Id)
		fmt.Printf("  Name:     %s\n", cred.Name)
		fmt.Printf("  Login:    %s\n", cred.Login)
		fmt.Printf("  Password: %s\n", decryptedPassword)

		if cred.Url != nil {
			fmt.Printf("  URL:      %s\n", *cred.Url)
		}
		if cred.Metadata != nil {
			fmt.Printf("  Metadata: %s\n", *cred.Metadata)
		}

		if cred.CreatedAt != nil {
			fmt.Printf("  Created:  %s\n", cred.CreatedAt.AsTime().Format(time.RFC3339))
		}
		if cred.UpdatedAt != nil {
			fmt.Printf("  Updated:  %s\n", cred.UpdatedAt.AsTime().Format(time.RFC3339))
		}

		return nil
	},
}

var credUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		name, _ := cmd.Flags().GetString("name")
		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")
		url, _ := cmd.Flags().GetString("url")
		metadata, _ := cmd.Flags().GetString("metadata")

		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("login") && !cmd.Flags().Changed("password") && !cmd.Flags().Changed("url") && !cmd.Flags().Changed("metadata") {
			return fmt.Errorf("at least one field must be specified for update")
		}

		// Инициализируем encryptor
		if err := ensureEncryptor(); err != nil {
			return err
		}

		// Get current credential to preserve unchanged fields
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		current, err := container.API.GetCredential(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get current credential: %w", err)
		}

		// Use current values if not specified
		if name == "" {
			name = current.Credential.Name
		}
		if login == "" {
			login = current.Credential.Login
		}

		// Для пароля нужно шифровать если он изменился
		encryptedPassword := current.Credential.Password // по умолчанию оставляем старый
		if password != "" {
			// Шифруем новый пароль
			encryptedPassword, err = container.Encryptor.Encrypt(password)
			if err != nil {
				return fmt.Errorf("failed to encrypt password: %w", err)
			}
		}

		req := &pb.CredentialUpdateRequest{
			Id:       id,
			Name:     name,
			Login:    login,
			Password: encryptedPassword,
		}

		if cmd.Flags().Changed("url") {
			if url != "" {
				req.Url = &url
			}
		} else {
			req.Url = current.Credential.Url
		}

		if cmd.Flags().Changed("metadata") {
			if metadata != "" {
				req.Metadata = &metadata
			}
		} else {
			req.Metadata = current.Credential.Metadata
		}

		resp, err := container.API.UpdateCredential(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to update credential: %w", err)
		}

		fmt.Printf("✓ Credential updated successfully!\n")
		fmt.Printf("  GetID:    %s\n", resp.Credential.Id)
		fmt.Printf("  Name:  %s\n", resp.Credential.Name)
		fmt.Printf("  Login: %s\n", resp.Credential.Login)

		return nil
	},
}

var credDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a credential",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to delete credential %s? (y/N): ", id)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := container.API.DeleteCredential(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete credential: %w", err)
		}

		if !resp.Deleted {
			return fmt.Errorf("credential was not deleted")
		}

		fmt.Printf("✓ Credential deleted successfully!\n")
		return nil
	},
}

// ensureEncryptor проверяет что encryptor инициализирован, если нет - запрашивает master password.
func ensureEncryptor() error {
	if container.Encryptor != nil {
		return nil
	}

	// Получаем текущего пользователя
	username, err := container.TokenStore.GetCurrentUsername()
	if err != nil || username == "" {
		return fmt.Errorf("no authenticated user found, please login first")
	}

	masterPassword, err := promptSecret("Master Password (for encryption)")
	if err != nil {
		return fmt.Errorf("failed to read master password: %w", err)
	}
	if masterPassword == "" {
		return fmt.Errorf("master password is required")
	}

	if err := container.InitEncryptor(username, masterPassword); err != nil {
		return fmt.Errorf("failed to init encryptor: %w", err)
	}

	return nil
}

func init() {
	credAddCmd.Flags().String("url", "", "URL associated with the credential")
	credAddCmd.Flags().String("metadata", "", "metadata for the credential (JSON or text)")

	credListCmd.Flags().Uint32("limit", 20, "number of items to return")
	credListCmd.Flags().Uint32("offset", 0, "offset for pagination")

	credUpdateCmd.Flags().String("name", "", "new name")
	credUpdateCmd.Flags().String("login", "", "new login")
	credUpdateCmd.Flags().String("password", "", "new password")
	credUpdateCmd.Flags().String("url", "", "new URL")
	credUpdateCmd.Flags().String("metadata", "", "new metadata")

	credDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	credCmd.AddCommand(credAddCmd)
	credCmd.AddCommand(credListCmd)
	credCmd.AddCommand(credGetCmd)
	credCmd.AddCommand(credUpdateCmd)
	credCmd.AddCommand(credDeleteCmd)
}
