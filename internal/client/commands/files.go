package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	pb "github.com/BigSm0uk/GophKeeper/pkg/proto/gophkeeper/v1"
	"github.com/spf13/cobra"
)

const chunkSize = 64 * 1024 // 64KB chunks

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "Manage binary files",
}

var fileUploadCmd = &cobra.Command{
	Use:   "upload <file-path>",
	Short: "Upload a binary file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		name, _ := cmd.Flags().GetString("name")
		metadata, _ := cmd.Flags().GetString("metadata")

		if name == "" {
			name = filepath.Base(filePath)
		}

		// Инициализируем encryptor
		if err := ensureEncryptor(); err != nil {
			return err
		}

		// Открываем файл
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()

		fileInfo, err := file.Stat()
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Создаем streaming клиент
		client := pb.NewBinariesServiceClient(container.API.GetConn())
		stream, err := client.UploadStream(ctx)
		if err != nil {
			return fmt.Errorf("failed to create upload stream: %w", err)
		}

		// Читаем и отправляем файл чанками
		buffer := make([]byte, chunkSize)
		totalSent := int64(0)

		for {
			n, err := file.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			// Шифруем чанк
			encryptedData, err := container.Encryptor.EncryptBytes(buffer[:n])
			if err != nil {
				return fmt.Errorf("failed to encrypt chunk: %w", err)
			}

			var chunk *pb.BinaryUploadChunk

			// Первый чанк содержит метаданные
			if totalSent == 0 {
				chunk = &pb.BinaryUploadChunk{
					Data: &pb.BinaryUploadChunk_Metadata{
						Metadata: &pb.BinaryMetadata{
							Name:        name,
							Filename:    filepath.Base(filePath),
							TotalSize:   fileInfo.Size(),
							ContentType: detectContentType(filePath),
							Metadata:    ptrStr(metadata),
						},
					},
				}
			} else {
				chunk = &pb.BinaryUploadChunk{
					Data: &pb.BinaryUploadChunk_ChunkData{
						ChunkData: encryptedData,
					},
				}
			}

			if err := stream.Send(chunk); err != nil {
				return fmt.Errorf("failed to send chunk: %w", err)
			}

			totalSent += int64(n)
			fmt.Printf("\rUploading: %.2f%%", float64(totalSent)/float64(fileInfo.Size())*100)
		}

		fmt.Println() // Новая строка после прогресса

		resp, err := stream.CloseAndRecv()
		if err != nil {
			return fmt.Errorf("failed to close stream: %w", err)
		}

		fmt.Printf("✅ File uploaded successfully!\n")
		fmt.Printf("  ID: %s\n", resp.Binary.Id)
		fmt.Printf("  Name: %s\n", resp.Binary.Name)
		fmt.Printf("  Size: %d bytes\n", resp.Binary.Size)

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

		// Инициализируем encryptor
		if err := ensureEncryptor(); err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Создаем streaming клиент
		client := pb.NewBinariesServiceClient(container.API.GetConn())
		stream, err := client.DownloadStream(ctx, &pb.BinaryDownloadRequest{
			Id: binaryID,
		})
		if err != nil {
			return fmt.Errorf("failed to create download stream: %w", err)
		}

		var outputFile *os.File
		var totalSize int64
		totalReceived := int64(0)
		firstChunk := true

		for {
			chunk, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				if outputFile != nil {
					outputFile.Close()
					os.Remove(outputFile.Name())
				}
				return fmt.Errorf("failed to receive chunk: %w", err)
			}

			// Первый чанк содержит total_size
			if firstChunk {
				totalSize = chunk.TotalSize
				firstChunk = false

				// Определяем путь для сохранения
				if outputPath == "" {
					outputPath = fmt.Sprintf("file_%s", binaryID)
				}

				// Создаем файл
				outputFile, err = os.Create(outputPath)
				if err != nil {
					return fmt.Errorf("failed to create output file: %w", err)
				}

				fmt.Printf("Downloading file (%d bytes)\n", totalSize)
			}

			if len(chunk.ChunkData) > 0 {
				// Дешифруем данные
				decryptedData, err := container.Encryptor.DecryptBytes(chunk.ChunkData)
				if err != nil {
					return fmt.Errorf("failed to decrypt chunk: %w", err)
				}

				_, err = outputFile.Write(decryptedData)
				if err != nil {
					return fmt.Errorf("failed to write to file: %w", err)
				}

				totalReceived += int64(len(decryptedData))
				if totalSize > 0 {
					fmt.Printf("\rProgress: %.2f%%", float64(totalReceived)/float64(totalSize)*100)
				}
			}
		}

		fmt.Println() // Новая строка после прогресса

		if outputFile != nil {
			outputFile.Close()
		}

		fmt.Printf("✅ File downloaded successfully to: %s\n", outputPath)
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
			fmt.Printf("Size:     %d bytes\n", binary.Size)
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
			fmt.Scanln(&confirm)
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

		fmt.Println("✅ File deleted")
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
