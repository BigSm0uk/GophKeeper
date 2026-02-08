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

var cardCmd = &cobra.Command{
	Use:   "card",
	Short: "Manage bank cards",
}

var cardAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new bank card",
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		// Инициализируем encryptor
		if err := ensureEncryptor(); err != nil {
			return err
		}

		name := promptRequired("Card Name")
		cardNumber := promptRequired("Card Number")
		cardholderName := promptRequired("Cardholder Name")
		expiryDate := promptRequired("Expiry Date (MM/YY)")
		cvv, err := promptSecret("CVV")
		if err != nil {
			return fmt.Errorf("failed to read CVV: %w", err)
		}
		if cvv == "" {
			return fmt.Errorf("CVV is required")
		}

		bankName, _ := cmd.Flags().GetString("bank")
		metadata, _ := cmd.Flags().GetString("metadata")

		// Шифруем чувствительные данные
		encryptedCardNumber, err := container.Encryptor.Encrypt(cardNumber)
		if err != nil {
			return fmt.Errorf("failed to encrypt card number: %w", err)
		}
		encryptedCVV, err := container.Encryptor.Encrypt(cvv)
		if err != nil {
			return fmt.Errorf("failed to encrypt CVV: %w", err)
		}
		encryptedExpiryDate, err := container.Encryptor.Encrypt(expiryDate)
		if err != nil {
			return fmt.Errorf("failed to encrypt expiry date: %w", err)
		}

		req := &pb.CardCreateRequest{
			Name:           name,
			CardNumber:     encryptedCardNumber,
			CardholderName: cardholderName,
			ExpiryDate:     encryptedExpiryDate,
			Cvv:            encryptedCVV,
		}
		if bankName != "" {
			req.BankName = &bankName
		}
		if metadata != "" {
			req.Metadata = &metadata
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := container.API.CreateCard(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to create card: %w", err)
		}

		fmt.Printf("✓ Card created successfully!\n")
		fmt.Printf("  GetID:          %s\n", resp.Card.Id)
		fmt.Printf("  Name:        %s\n", resp.Card.Name)
		fmt.Printf("  Holder:      %s\n", resp.Card.CardholderName)
		fmt.Printf("  Number:      ****%s\n", resp.Card.CardNumber[len(resp.Card.CardNumber)-4:])

		return nil
	},
}

var cardListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all bank cards",
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

		resp, err := container.API.ListCards(ctx, limit, offset)
		if err != nil {
			return fmt.Errorf("failed to list cards: %w", err)
		}

		if len(resp.Items) == 0 {
			fmt.Println("No cards found.")
			return nil
		}

		fmt.Printf("Total cards: %d\n\n", resp.Page.Total)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "GetID\tNAME\tHOLDER\tNUMBER\tEXPIRY\tUPDATED")
		fmt.Fprintln(w, "--\t----\t------\t------\t------\t-------")

		for _, item := range resp.Items {
			updatedAt := "N/A"
			if item.UpdatedAt != nil {
				updatedAt = item.UpdatedAt.AsTime().Format("2006-01-02 15:04")
			}

			idShort := item.Id
			if len(idShort) > 8 {
				idShort = idShort[:8]
			}

			// Mask card number
			masked := "****" + item.CardNumber[len(item.CardNumber)-4:]

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				idShort, item.Name, item.CardholderName, masked, item.ExpiryDate, updatedAt)
		}
		w.Flush()

		if offset+limit < resp.Page.Total {
			fmt.Printf("\nShowing %d-%d of %d. Use --offset to see more.\n", offset+1, offset+uint32(len(resp.Items)), resp.Page.Total)
		}

		return nil
	},
}

var cardGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a card by GetID",
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

		resp, err := container.API.GetCard(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get card: %w", err)
		}

		card := resp.Card

		// Дешифруем чувствительные данные
		decryptedCardNumber, err := container.Encryptor.Decrypt(card.CardNumber)
		if err != nil {
			return fmt.Errorf("failed to decrypt card number: %w", err)
		}
		decryptedExpiryDate, err := container.Encryptor.Decrypt(card.ExpiryDate)
		if err != nil {
			return fmt.Errorf("failed to decrypt expiry date: %w", err)
		}
		decryptedCVV, err := container.Encryptor.Decrypt(card.Cvv)
		if err != nil {
			return fmt.Errorf("failed to decrypt CVV: %w", err)
		}

		fmt.Printf("Card Details:\n")
		fmt.Printf("  GetID:          %s\n", card.Id)
		fmt.Printf("  Name:        %s\n", card.Name)
		fmt.Printf("  Cardholder:  %s\n", card.CardholderName)
		fmt.Printf("  Number:      %s\n", decryptedCardNumber)
		fmt.Printf("  Expiry:      %s\n", decryptedExpiryDate)
		fmt.Printf("  CVV:         %s\n", decryptedCVV)

		if card.BankName != nil {
			fmt.Printf("  Bank:        %s\n", *card.BankName)
		}
		if card.Metadata != nil {
			fmt.Printf("  Metadata:    %s\n", *card.Metadata)
		}

		if card.CreatedAt != nil {
			fmt.Printf("  Created:     %s\n", card.CreatedAt.AsTime().Format(time.RFC3339))
		}
		if card.UpdatedAt != nil {
			fmt.Printf("  Updated:     %s\n", card.UpdatedAt.AsTime().Format(time.RFC3339))
		}

		return nil
	},
}

var cardUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a card",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		name, _ := cmd.Flags().GetString("name")
		cardNumber, _ := cmd.Flags().GetString("number")
		cardholderName, _ := cmd.Flags().GetString("holder")
		expiryDate, _ := cmd.Flags().GetString("expiry")
		cvv, _ := cmd.Flags().GetString("cvv")
		bankName, _ := cmd.Flags().GetString("bank")
		metadata, _ := cmd.Flags().GetString("metadata")

		if !cmd.Flags().Changed("name") && !cmd.Flags().Changed("number") && !cmd.Flags().Changed("holder") && !cmd.Flags().Changed("expiry") && !cmd.Flags().Changed("cvv") && !cmd.Flags().Changed("bank") && !cmd.Flags().Changed("metadata") {
			return fmt.Errorf("at least one field must be specified for update")
		}

		// Get current card to preserve unchanged fields
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		current, err := container.API.GetCard(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get current card: %w", err)
		}

		// Use current values if not specified
		if name == "" {
			name = current.Card.Name
		}
		if cardNumber == "" {
			cardNumber = current.Card.CardNumber
		}
		if cardholderName == "" {
			cardholderName = current.Card.CardholderName
		}
		if expiryDate == "" {
			expiryDate = current.Card.ExpiryDate
		}
		if cvv == "" {
			cvv = current.Card.Cvv
		}

		req := &pb.CardUpdateRequest{
			Id:             id,
			Name:           name,
			CardNumber:     cardNumber,
			CardholderName: cardholderName,
			ExpiryDate:     expiryDate,
			Cvv:            cvv,
		}

		if cmd.Flags().Changed("bank") {
			if bankName != "" {
				req.BankName = &bankName
			}
		} else {
			req.BankName = current.Card.BankName
		}

		if cmd.Flags().Changed("metadata") {
			if metadata != "" {
				req.Metadata = &metadata
			}
		} else {
			req.Metadata = current.Card.Metadata
		}

		resp, err := container.API.UpdateCard(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to update card: %w", err)
		}

		fmt.Printf("✓ Card updated successfully!\n")
		fmt.Printf("  GetID:          %s\n", resp.Card.Id)
		fmt.Printf("  Name:        %s\n", resp.Card.Name)
		fmt.Printf("  Cardholder:  %s\n", resp.Card.CardholderName)

		return nil
	},
}

var cardDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a card",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if container == nil {
			return fmt.Errorf("client not initialized")
		}

		id := args[0]

		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to delete card %s? (y/N): ", id)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := container.API.DeleteCard(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete card: %w", err)
		}

		if !resp.Deleted {
			return fmt.Errorf("card was not deleted")
		}

		fmt.Printf("✓ Card deleted successfully!\n")
		return nil
	},
}

func init() {
	cardAddCmd.Flags().String("bank", "", "bank name")
	cardAddCmd.Flags().String("metadata", "", "metadata for the card (JSON or text)")

	cardListCmd.Flags().Uint32("limit", 20, "number of items to return")
	cardListCmd.Flags().Uint32("offset", 0, "offset for pagination")

	cardUpdateCmd.Flags().String("name", "", "new name")
	cardUpdateCmd.Flags().String("number", "", "new card number")
	cardUpdateCmd.Flags().String("holder", "", "new cardholder name")
	cardUpdateCmd.Flags().String("expiry", "", "new expiry date (MM/YY)")
	cardUpdateCmd.Flags().String("cvv", "", "new CVV")
	cardUpdateCmd.Flags().String("bank", "", "new bank name")
	cardUpdateCmd.Flags().String("metadata", "", "new metadata")

	cardDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	cardCmd.AddCommand(cardAddCmd)
	cardCmd.AddCommand(cardListCmd)
	cardCmd.AddCommand(cardGetCmd)
	cardCmd.AddCommand(cardUpdateCmd)
	cardCmd.AddCommand(cardDeleteCmd)
}
