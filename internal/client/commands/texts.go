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

var textCmd = &cobra.Command{
	Use:   "text",
	Short: "Manage text notes",
}

var textAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new text note",
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		if err := ensureEncryptor(); err != nil {
			return err
		}

		name := promptRequired("Name")
		content := promptRequired("Content")
		metadata, _ := cmd.Flags().GetString("metadata")

		var metaPtr *string
		if metadata != "" {
			metaPtr = &metadata
		}

		offlineSvc := getOfflineService()

		if isOnline() {
			encryptedContent, encErr := container.Encryptor.Encrypt(content)
			if encErr != nil {
				return fmt.Errorf("failed to encrypt content: %w", encErr)
			}

			req := &pb.TextCreateRequest{
				Name:     name,
				Content:  encryptedContent,
				Metadata: metaPtr,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, apiErr := container.API.CreateText(ctx, req)
			if apiErr == nil {
				fmt.Printf("✓ Text note created on server!\n")
				fmt.Printf("  ID:      %s\n", resp.Text.Id)
				fmt.Printf("  Name:    %s\n", resp.Text.Name)
				fmt.Printf("  Content: %s\n", truncate(resp.Text.Content, 50))

				if offlineSvc != nil {
					_, _ = offlineSvc.CreateText(ctx, name, content, metaPtr)
				}
				return nil
			}
			fmt.Printf("⚠ Server unavailable, saving locally: %v\n", apiErr)
		}

		if offlineSvc == nil {
			return fmt.Errorf("offline storage not initialized")
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		text, err := offlineSvc.CreateText(ctx2, name, content, metaPtr)
		if err != nil {
			return fmt.Errorf("failed to save text locally: %w", err)
		}

		fmt.Printf("✓ Text note saved locally (will sync when online)\n")
		fmt.Printf("  ID:      %s\n", text.ID)
		fmt.Printf("  Name:    %s\n", text.Name)
		fmt.Printf("  Content: %s\n", truncate(text.Content, 50))

		return nil
	},
}

var textListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all text notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		limit, _ := cmd.Flags().GetUint32("limit")
		offset, _ := cmd.Flags().GetUint32("offset")

		if limit == 0 {
			limit = 20
		}

		if isOnline() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := container.API.ListTexts(ctx, limit, offset)
			if err == nil {
				if len(resp.Items) == 0 {
					fmt.Println("No text notes found.")
					return nil
				}

				fmt.Printf("Total text notes: %d\n\n", resp.Page.Total)
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "ID\tNAME\tCONTENT\tUPDATED")
				fmt.Fprintln(w, "--\t----\t-------\t-------")

				for _, item := range resp.Items {
					updatedAt := "N/A"
					if item.UpdatedAt != nil {
						updatedAt = item.UpdatedAt.AsTime().Format("2006-01-02 15:04")
					}
					idShort := item.Id
					if len(idShort) > 8 {
						idShort = idShort[:8]
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
						idShort, item.Name, truncate(item.Content, 40), updatedAt)
				}
				w.Flush()

				if offset+limit < resp.Page.Total {
					fmt.Printf("\nShowing %d-%d of %d. Use --offset to see more.\n",
						offset+1, offset+uint32(len(resp.Items)), resp.Page.Total)
				}
				return nil
			}
			fmt.Printf("⚠ Server unavailable, showing local data: %v\n", err)
		}

		// Offline
		offlineSvc := getOfflineService()
		if offlineSvc == nil {
			return fmt.Errorf("offline storage not initialized")
		}

		if err := ensureEncryptor(); err != nil {
			return err
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		texts, err := offlineSvc.ListTexts(ctx2)
		if err != nil {
			return fmt.Errorf("failed to list local texts: %w", err)
		}

		if len(texts) == 0 {
			fmt.Println("No text notes found (offline mode).")
			return nil
		}

		fmt.Printf("Offline mode - showing %d local text notes:\n\n", len(texts))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tCONTENT\tSTATUS\tUPDATED")
		fmt.Fprintln(w, "--\t----\t-------\t------\t-------")

		for _, text := range texts {
			idShort := text.ID
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				idShort, text.Name, truncate(text.Content, 30), string(text.SyncStatus),
				text.UpdatedAt.Format("2006-01-02 15:04"))
		}
		w.Flush()

		return nil
	},
}

var textGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a text note by ID",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		if err := ensureEncryptor(); err != nil {
			return err
		}

		// Try online first
		if isOnline() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := container.API.GetText(ctx, id)
			if err == nil {
				text := resp.Text

				decryptedContent, decErr := container.Encryptor.Decrypt(text.Content)
				if decErr != nil {
					return fmt.Errorf("failed to decrypt content: %w", decErr)
				}

				fmt.Printf("Text Note Details:\n")
				fmt.Printf("  ID:      %s\n", text.Id)
				fmt.Printf("  Name:    %s\n", text.Name)
				fmt.Printf("  Content:\n%s\n", decryptedContent)

				if text.Metadata != nil {
					fmt.Printf("  Metadata: %s\n", *text.Metadata)
				}

				if text.CreatedAt != nil {
					fmt.Printf("  Created:  %s\n", text.CreatedAt.AsTime().Format(time.RFC3339))
				}
				if text.UpdatedAt != nil {
					fmt.Printf("  Updated:  %s\n", text.UpdatedAt.AsTime().Format(time.RFC3339))
				}

				return nil
			}
			fmt.Printf("⚠ Server unavailable, showing local data: %v\n", err)
		}

		// Offline: get from local
		offlineSvc := getOfflineService()
		if offlineSvc == nil {
			return fmt.Errorf("offline storage not initialized")
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		text, err := offlineSvc.GetText(ctx2, id)
		if err != nil {
			return fmt.Errorf("failed to get text locally: %w", err)
		}

		fmt.Printf("Text Note Details (offline):\n")
		fmt.Printf("  ID:      %s\n", text.ID)
		fmt.Printf("  Name:    %s\n", text.Name)
		fmt.Printf("  Content:\n%s\n", text.Content)
		if text.Metadata != nil {
			fmt.Printf("  Metadata: %s\n", *text.Metadata)
		}
		fmt.Printf("  Status:  %s\n", text.SyncStatus)
		fmt.Printf("  Updated: %s\n", text.UpdatedAt.Format(time.RFC3339))

		return nil
	},
}

var textUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a text note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		name, _ := cmd.Flags().GetString("name")
		content, _ := cmd.Flags().GetString("content")
		metadata, _ := cmd.Flags().GetString("metadata")

		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("content") && !cmd.Flags().Changed("metadata") {
			return fmt.Errorf("at least one field must be specified for update")
		}

		if err := ensureEncryptor(); err != nil {
			return err
		}

		// Try online first
		if isOnline() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			current, err := container.API.GetText(ctx, id)
			if err == nil {
				if name == "" {
					name = current.Text.Name
				}
				if content == "" {
					content = current.Text.Content
				}

				req := &pb.TextUpdateRequest{
					Id:      id,
					Name:    name,
					Content: content,
				}

				if cmd.Flags().Changed("metadata") {
					if metadata != "" {
						req.Metadata = &metadata
					}
				} else {
					req.Metadata = current.Text.Metadata
				}

				resp, apiErr := container.API.UpdateText(ctx, req)
				if apiErr == nil {
					fmt.Printf("✓ Text note updated successfully!\n")
					fmt.Printf("  ID:      %s\n", resp.Text.Id)
					fmt.Printf("  Name:    %s\n", resp.Text.Name)
					fmt.Printf("  Content: %s\n", truncate(resp.Text.Content, 50))
					return nil
				}
				fmt.Printf("⚠ Server unavailable for update, saving locally: %v\n", apiErr)
			} else {
				fmt.Printf("⚠ Server unavailable, updating locally: %v\n", err)
			}
		}

		// Offline: update locally
		offlineSvc := getOfflineService()
		if offlineSvc == nil {
			return fmt.Errorf("offline storage not initialized")
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		existing, err := offlineSvc.GetText(ctx2, id)
		if err != nil {
			return fmt.Errorf("text not found locally: %w", err)
		}

		if name == "" {
			name = existing.Name
		}
		if content == "" {
			content = existing.Content
		}

		var metaPtr *string
		if cmd.Flags().Changed("metadata") {
			if metadata != "" {
				metaPtr = &metadata
			}
		} else {
			metaPtr = existing.Metadata
		}

		updated, err := offlineSvc.UpdateText(ctx2, id, name, content, metaPtr)
		if err != nil {
			return fmt.Errorf("failed to update text locally: %w", err)
		}

		fmt.Printf("✓ Text note updated locally (will sync when online)\n")
		fmt.Printf("  ID:      %s\n", updated.ID)
		fmt.Printf("  Name:    %s\n", updated.Name)
		fmt.Printf("  Content: %s\n", truncate(updated.Content, 50))

		return nil
	},
}

var textDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a text note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to delete text note %s? (y/N): ", id)
			var response string
			_, _ = fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		if isOnline() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			resp, err := container.API.DeleteText(ctx, id)
			if err == nil {
				if !resp.Deleted {
					return fmt.Errorf("text was not deleted")
				}
				offlineSvc := getOfflineService()
				if offlineSvc != nil {
					_ = offlineSvc.DeleteText(ctx, id)
				}
				fmt.Printf("✓ Text note deleted successfully!\n")
				return nil
			}
			fmt.Printf("⚠ Server unavailable, deleting locally: %v\n", err)
		}

		offlineSvc := getOfflineService()
		if offlineSvc == nil {
			return fmt.Errorf("offline storage not initialized")
		}

		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()

		if err := offlineSvc.DeleteText(ctx2, id); err != nil {
			return fmt.Errorf("failed to delete text locally: %w", err)
		}

		fmt.Printf("✓ Text note marked for deletion (will sync when online)\n")
		return nil
	},
}

func init() {
	textAddCmd.Flags().String("metadata", "", "metadata for the text note (JSON or text)")

	textListCmd.Flags().Uint32("limit", 20, "number of items to return")
	textListCmd.Flags().Uint32("offset", 0, "offset for pagination")

	textUpdateCmd.Flags().String("name", "", "new name")
	textUpdateCmd.Flags().String("content", "", "new content")
	textUpdateCmd.Flags().String("metadata", "", "new metadata")

	textDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	textCmd.AddCommand(textAddCmd)
	textCmd.AddCommand(textListCmd)
	textCmd.AddCommand(textGetCmd)
	textCmd.AddCommand(textUpdateCmd)
	textCmd.AddCommand(textDeleteCmd)
}

// truncate returns truncated string with "..." if it exceeds maxLen
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
