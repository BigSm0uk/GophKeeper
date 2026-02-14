package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/BigSm0uk/GophKeeper/internal/client/crypto"
	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/spf13/cobra"
)

const (
	uploadChunkSize = 64 * 1024 // 64KB gRPC chunks
	// Dynamic timeout: base + rate per 100MB
	baseUploadTimeout = 5 * time.Minute
	timeoutPer100MB   = 1 * time.Minute
)

// dynamicTimeout computes a context timeout based on file size.
func dynamicTimeout(fileSize int64) time.Duration {
	extra := time.Duration(fileSize/(100*1024*1024)) * timeoutPer100MB
	timeout := baseUploadTimeout + extra
	if timeout < baseUploadTimeout {
		timeout = baseUploadTimeout
	}
	return timeout
}

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Manage binary files",
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload <file-path>",
	Short: "Upload a binary file (supports files of any size)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		name, _ := cmd.Flags().GetString("name")
		metadata, _ := cmd.Flags().GetString("metadata")

		if name == "" {
			name = filepath.Base(filePath)
		}

		if err := ensureEncryptor(); err != nil {
			return err
		}

		// Verify source file exists and get its size
		srcInfo, err := os.Stat(filePath)
		if err != nil {
			return fmt.Errorf("failed to stat file: %w", err)
		}
		originalSize := srcInfo.Size()

		fmt.Printf("Encrypting file (%s, %s)...\n", filepath.Base(filePath), formatSize(originalSize))

		// Phase 1: Encrypt file locally using streaming encryption
		se, err := crypto.NewStreamEncryptorFromEncryptor(container.Encryptor)
		if err != nil {
			return fmt.Errorf("failed to create stream encryptor: %w", err)
		}

		tmpEncPath := filepath.Join(os.TempDir(), fmt.Sprintf("gophkeeper-upload-%d.enc", time.Now().UnixNano()))
		defer os.Remove(tmpEncPath)

		encryptedSize, encryptedChecksum, err := se.EncryptFile(filePath, tmpEncPath)
		if err != nil {
			return fmt.Errorf("failed to encrypt file: %w", err)
		}

		fmt.Printf("Encrypted: %s (checksum: %s...)\n", formatSize(encryptedSize), encryptedChecksum[:16])

		// Phase 2: Stream encrypted file to server
		timeout := dynamicTimeout(encryptedSize)
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		client := pb.NewBinariesServiceClient(container.API.GetConn())
		stream, err := client.UploadStream(ctx)
		if err != nil {
			return fmt.Errorf("failed to create upload stream: %w", err)
		}

		// Send metadata first
		metaChunk := &pb.BinaryUploadChunk{
			Data: &pb.BinaryUploadChunk_Metadata{
				Metadata: &pb.BinaryMetadata{
					Name:        name,
					Filename:    filepath.Base(filePath),
					TotalSize:   encryptedSize,
					ContentType: detectContentType(filePath),
					Metadata:    ptrStr(metadata),
					Checksum:    encryptedChecksum,
				},
			},
		}
		if err := stream.Send(metaChunk); err != nil {
			return fmt.Errorf("failed to send metadata: %w", err)
		}

		// Stream encrypted file in chunks
		encFile, err := os.Open(tmpEncPath)
		if err != nil {
			return fmt.Errorf("failed to open encrypted file: %w", err)
		}
		defer encFile.Close()

		buf := make([]byte, uploadChunkSize)
		totalSent := int64(0)

		for {
			n, readErr := encFile.Read(buf)
			if n > 0 {
				chunk := &pb.BinaryUploadChunk{
					Data: &pb.BinaryUploadChunk_ChunkData{
						ChunkData: buf[:n],
					},
				}
				if err := stream.Send(chunk); err != nil {
					return fmt.Errorf("failed to send chunk: %w", err)
				}
				totalSent += int64(n)
				fmt.Printf("\rUploading: %.2f%%", float64(totalSent)/float64(encryptedSize)*100)
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return fmt.Errorf("failed to read encrypted file: %w", readErr)
			}
		}

		fmt.Println()

		resp, err := stream.CloseAndRecv()
		if err != nil {
			return fmt.Errorf("upload failed: %w", err)
		}

		fmt.Printf("File uploaded successfully!\n")
		fmt.Printf("  ID:   %s\n", resp.Binary.Id)
		fmt.Printf("  Name: %s\n", resp.Binary.Name)
		fmt.Printf("  Size: %s\n", formatSize(resp.Binary.Size))

		return nil
	},
}

var fileDownloadCmd = &cobra.Command{
	Use:   "download <binary-id> [output-path]",
	Short: "Download a binary file",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		binaryID := args[0]
		outputPath := ""
		if len(args) > 1 {
			outputPath = args[1]
		}

		if err := ensureEncryptor(); err != nil {
			return err
		}

		// Phase 1: Download encrypted data to a temp file
		fmt.Println("Downloading encrypted file...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		client := pb.NewBinariesServiceClient(container.API.GetConn())
		stream, err := client.DownloadStream(ctx, &pb.BinaryDownloadRequest{
			Id: binaryID,
		})
		if err != nil {
			return fmt.Errorf("failed to create download stream: %w", err)
		}

		tmpEncPath := filepath.Join(os.TempDir(), fmt.Sprintf("gophkeeper-download-%d.enc", time.Now().UnixNano()))
		tmpFile, err := os.Create(tmpEncPath)
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}

		var totalSize int64
		totalReceived := int64(0)
		firstChunk := true

		for {
			chunk, recvErr := stream.Recv()
			if recvErr == io.EOF {
				break
			}
			if recvErr != nil {
				tmpFile.Close()
				os.Remove(tmpEncPath)
				return fmt.Errorf("failed to receive chunk: %w", recvErr)
			}

			if firstChunk {
				totalSize = chunk.TotalSize
				firstChunk = false
				fmt.Printf("File size: %s\n", formatSize(totalSize))
			}

			if len(chunk.ChunkData) > 0 {
				if _, err := tmpFile.Write(chunk.ChunkData); err != nil {
					tmpFile.Close()
					os.Remove(tmpEncPath)
					return fmt.Errorf("failed to write chunk: %w", err)
				}
				totalReceived += int64(len(chunk.ChunkData))
				if totalSize > 0 {
					fmt.Printf("\rDownloading: %.2f%%", float64(totalReceived)/float64(totalSize)*100)
				}
			}
		}
		tmpFile.Close()
		fmt.Println()

		defer os.Remove(tmpEncPath)

		// Phase 2: Decrypt the downloaded file
		if outputPath == "" {
			outputPath = fmt.Sprintf("file_%s", binaryID)
		}

		fmt.Println("Decrypting file...")

		se, err := crypto.NewStreamEncryptorFromEncryptor(container.Encryptor)
		if err != nil {
			return fmt.Errorf("failed to create stream encryptor: %w", err)
		}

		if err := se.DecryptFile(tmpEncPath, outputPath); err != nil {
			// On failure: delete incomplete output file completely
			os.Remove(outputPath)
			return fmt.Errorf("failed to decrypt file: %w", err)
		}

		fmt.Printf("File downloaded successfully to: %s\n", outputPath)
		return nil
	},
}

var fileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List binary files",
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetUint32("limit")
		offset, _ := cmd.Flags().GetUint32("offset")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := pb.NewBinariesServiceClient(container.API.GetConn())
		resp, err := client.List(ctx, &pb.BinaryListRequest{
			Page: &pb.PageRequest{
				Limit:  limit,
				Offset: offset,
			},
		})
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		if len(resp.Items) == 0 {
			fmt.Println("No files found")
			return nil
		}

		fmt.Printf("Binary Files (Total: %d)\n\n", resp.Page.Total)
		for _, binary := range resp.Items {
			fmt.Printf("ID:       %s\n", binary.Id)
			fmt.Printf("Name:     %s\n", binary.Name)
			fmt.Printf("Filename: %s\n", binary.Filename)
			fmt.Printf("Size:     %s\n", formatSize(binary.Size))
			fmt.Printf("Type:     %s\n", binary.ContentType)
			fmt.Printf("Created:  %s\n", binary.CreatedAt.AsTime().Format(time.RFC3339))
			fmt.Println("---")
		}

		return nil
	},
}

var fileDeleteCmd = &cobra.Command{
	Use:   "delete <binary-id>",
	Short: "Delete a binary file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		binaryID := args[0]
		force, _ := cmd.Flags().GetBool("force")

		if !force {
			fmt.Printf("Are you sure you want to delete file %s? (y/N): ", binaryID)
			var confirm string
			_, _ = fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := pb.NewBinariesServiceClient(container.API.GetConn())
		_, err := client.Delete(ctx, &pb.BinaryDeleteRequest{
			Id: binaryID,
		})
		if err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}

		fmt.Println("File deleted")
		return nil
	},
}

func init() {
	fileUploadCmd.Flags().String("name", "", "display name for the file")
	fileUploadCmd.Flags().String("metadata", "", "metadata for the file (JSON or text)")

	fileListCmd.Flags().Uint32("limit", 20, "number of items to return")
	fileListCmd.Flags().Uint32("offset", 0, "offset for pagination")

	fileDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation")

	filesCmd.AddCommand(fileUploadCmd)
	filesCmd.AddCommand(fileDownloadCmd)
	filesCmd.AddCommand(fileListCmd)
	filesCmd.AddCommand(fileDeleteCmd)

	rootCmd.AddCommand(filesCmd)
}

func detectContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/float64(TB))
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}
