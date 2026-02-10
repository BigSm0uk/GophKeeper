package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
)

const (
	// StreamMagic is the magic number for encrypted file format.
	StreamMagic = "GKFS"
	// StreamVersion is the current format version.
	StreamVersion = byte(0x01)
	// DefaultChunkSize is the default plaintext chunk size (1MB).
	DefaultChunkSize = 1 << 20 // 1 MiB
	// StreamHeaderSize is the fixed header size: magic(4) + version(1) + chunkSize(4) + originalSize(8) + origChecksumPrefix(8).
	StreamHeaderSize = 25
	// gcmNonceSize is the standard GCM nonce size.
	gcmNonceSize = 12
	// gcmTagSize is the standard GCM authentication tag size.
	gcmTagSize = 16
	// gcmOverhead is the per-chunk overhead (nonce + tag).
	gcmOverhead = gcmNonceSize + gcmTagSize
)

// StreamEncryptor provides streaming file encryption/decryption using chunked AES-256-GCM.
// Each chunk is encrypted independently with a counter-derived nonce, allowing constant
// memory usage regardless of file size.
type StreamEncryptor struct {
	key       []byte
	chunkSize int
}

// NewStreamEncryptor creates a new StreamEncryptor with the given encryption key.
func NewStreamEncryptor(key []byte) (*StreamEncryptor, error) {
	if len(key) != KeyLen {
		return nil, fmt.Errorf("key must be %d bytes, got %d", KeyLen, len(key))
	}
	return &StreamEncryptor{key: key, chunkSize: DefaultChunkSize}, nil
}

// NewStreamEncryptorFromEncryptor creates a StreamEncryptor reusing the key from an Encryptor.
func NewStreamEncryptorFromEncryptor(enc *Encryptor) (*StreamEncryptor, error) {
	if enc == nil {
		return nil, fmt.Errorf("encryptor is nil")
	}
	return NewStreamEncryptor(enc.key)
}

// EncryptFile encrypts srcPath to dstPath using streaming chunked AES-256-GCM.
// Returns the encrypted file size and SHA256 checksum of encrypted data.
func (se *StreamEncryptor) EncryptFile(srcPath, dstPath string) (encryptedSize int64, encryptedChecksum string, err error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return 0, "", fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	srcInfo, err := src.Stat()
	if err != nil {
		return 0, "", fmt.Errorf("stat source file: %w", err)
	}
	originalSize := srcInfo.Size()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, "", fmt.Errorf("create destination file: %w", err)
	}
	defer func() {
		dst.Close()
		if err != nil {
			os.Remove(dstPath)
		}
	}()

	// Phase 1: encrypt all chunks, tracking original file checksum
	origHasher := sha256.New()
	teeReader := io.TeeReader(src, origHasher)

	encryptedSize, err = se.encryptToWriter(teeReader, dst, originalSize)
	if err != nil {
		return 0, "", err
	}

	// Phase 2: patch header with original checksum prefix
	origSum := origHasher.Sum(nil)
	if _, err = dst.Seek(int64(StreamHeaderSize-8), io.SeekStart); err != nil {
		return 0, "", fmt.Errorf("seek to patch header: %w", err)
	}
	if _, err = dst.Write(origSum[:8]); err != nil {
		return 0, "", fmt.Errorf("write checksum prefix: %w", err)
	}

	// Phase 3: compute SHA256 of the complete encrypted file
	if _, err = dst.Seek(0, io.SeekStart); err != nil {
		return 0, "", fmt.Errorf("seek to start for checksum: %w", err)
	}
	encHasher := sha256.New()
	if _, err = io.Copy(encHasher, dst); err != nil {
		return 0, "", fmt.Errorf("compute encrypted checksum: %w", err)
	}

	encryptedChecksum = hex.EncodeToString(encHasher.Sum(nil))
	return encryptedSize, encryptedChecksum, nil
}

// EncryptStream encrypts data from reader to writer using streaming chunked AES-256-GCM.
// originalSize must be the exact size of the plaintext data in the reader.
// Returns the total bytes written and SHA256 checksum of encrypted output.
// NOTE: The origChecksumPrefix in the header will be zeroes (stream mode does not patch).
func (se *StreamEncryptor) EncryptStream(reader io.Reader, writer io.Writer, originalSize int64) (int64, string, error) {
	encHasher := sha256.New()
	teeWriter := io.MultiWriter(writer, encHasher)

	totalWritten, err := se.encryptToWriter(reader, teeWriter, originalSize)
	if err != nil {
		return 0, "", err
	}

	checksum := hex.EncodeToString(encHasher.Sum(nil))
	return totalWritten, checksum, nil
}

// encryptToWriter encrypts data from reader to writer (core logic without checksum).
func (se *StreamEncryptor) encryptToWriter(reader io.Reader, writer io.Writer, originalSize int64) (int64, error) {
	block, err := aes.NewCipher(se.key)
	if err != nil {
		return 0, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return 0, fmt.Errorf("create GCM: %w", err)
	}

	// Write header (origChecksumPrefix zeroed, caller may patch later)
	header := se.buildHeader(originalSize, nil)
	headerN, err := writer.Write(header)
	if err != nil {
		return 0, fmt.Errorf("write header: %w", err)
	}
	totalWritten := int64(headerN)

	// Encrypt chunks
	buf := make([]byte, se.chunkSize)
	chunkIndex := uint64(0)

	for {
		n, readErr := io.ReadFull(reader, buf)
		if n > 0 {
			nonce := counterNonce(chunkIndex, gcm.NonceSize())
			encrypted := gcm.Seal(nonce, nonce, buf[:n], nil)

			written, writeErr := writer.Write(encrypted)
			if writeErr != nil {
				return totalWritten, fmt.Errorf("write encrypted chunk %d: %w", chunkIndex, writeErr)
			}
			totalWritten += int64(written)
			chunkIndex++
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return totalWritten, fmt.Errorf("read chunk %d: %w", chunkIndex, readErr)
		}
	}

	return totalWritten, nil
}

// DecryptFile decrypts srcPath to dstPath using streaming chunked AES-256-GCM.
func (se *StreamEncryptor) DecryptFile(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open encrypted file: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer func() {
		dst.Close()
		if err != nil {
			os.Remove(dstPath)
		}
	}()

	err = se.DecryptStream(src, dst)
	return err
}

// DecryptStream decrypts data from reader to writer using streaming chunked AES-256-GCM.
func (se *StreamEncryptor) DecryptStream(reader io.Reader, writer io.Writer) error {
	block, err := aes.NewCipher(se.key)
	if err != nil {
		return fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("create GCM: %w", err)
	}

	// Read and validate header
	header := make([]byte, StreamHeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return fmt.Errorf("read header: %w", err)
	}

	chunkSize, _, err := parseHeader(header)
	if err != nil {
		return fmt.Errorf("invalid header: %w", err)
	}

	// Each encrypted chunk = nonce(12) + ciphertext(<=chunkSize) + tag(16)
	// Max encrypted chunk size
	maxEncChunkSize := gcmNonceSize + chunkSize + gcmTagSize
	buf := make([]byte, maxEncChunkSize)
	chunkIndex := uint64(0)

	for {
		n, readErr := io.ReadAtLeast(reader, buf, gcmOverhead+1)
		if n > 0 {
			expectedNonce := counterNonce(chunkIndex, gcm.NonceSize())
			actualNonce := buf[:gcm.NonceSize()]

			// Validate nonce matches expected counter
			for i := range expectedNonce {
				if expectedNonce[i] != actualNonce[i] {
					return fmt.Errorf("nonce mismatch at chunk %d: possible tampering or reordering", chunkIndex)
				}
			}

			plaintext, decErr := gcm.Open(nil, actualNonce, buf[gcm.NonceSize():n], nil)
			if decErr != nil {
				return fmt.Errorf("decrypt chunk %d: %w", chunkIndex, decErr)
			}

			if _, writeErr := writer.Write(plaintext); writeErr != nil {
				return fmt.Errorf("write decrypted chunk %d: %w", chunkIndex, writeErr)
			}
			chunkIndex++
		}

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read encrypted chunk %d: %w", chunkIndex, readErr)
		}
	}

	return nil
}

// EncryptedFileSize calculates the total encrypted file size for a given plaintext size.
func (se *StreamEncryptor) EncryptedFileSize(originalSize int64) int64 {
	if originalSize == 0 {
		return int64(StreamHeaderSize)
	}
	numFullChunks := originalSize / int64(se.chunkSize)
	remainder := originalSize % int64(se.chunkSize)

	total := int64(StreamHeaderSize)
	total += numFullChunks * int64(se.chunkSize+gcmOverhead)
	if remainder > 0 {
		total += remainder + int64(gcmOverhead)
	}
	return total
}

// GetOriginalSize reads only the header of an encrypted file and returns the original plaintext size.
func (se *StreamEncryptor) GetOriginalSize(encryptedPath string) (int64, error) {
	f, err := os.Open(encryptedPath)
	if err != nil {
		return 0, fmt.Errorf("open encrypted file: %w", err)
	}
	defer f.Close()

	header := make([]byte, StreamHeaderSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}

	_, originalSize, err := parseHeader(header)
	if err != nil {
		return 0, err
	}
	return originalSize, nil
}

// ComputeFileChecksum computes SHA256 of the given file path.
func ComputeFileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ComputeReaderChecksum computes SHA256 while reading and writing to a hasher.
func ComputeReaderChecksum(reader io.Reader) (hash.Hash, io.Reader) {
	h := sha256.New()
	return h, io.TeeReader(reader, h)
}

// buildHeader creates the file header.
func (se *StreamEncryptor) buildHeader(originalSize int64, origChecksumPrefix []byte) []byte {
	header := make([]byte, StreamHeaderSize)
	copy(header[0:4], StreamMagic)
	header[4] = StreamVersion
	binary.LittleEndian.PutUint32(header[5:9], uint32(se.chunkSize))
	binary.LittleEndian.PutUint64(header[9:17], uint64(originalSize))
	if len(origChecksumPrefix) >= 8 {
		copy(header[17:25], origChecksumPrefix[:8])
	}
	return header
}

// parseHeader validates and parses the file header.
// Returns chunkSize, originalSize.
func parseHeader(header []byte) (int, int64, error) {
	if len(header) < StreamHeaderSize {
		return 0, 0, fmt.Errorf("header too short: %d bytes", len(header))
	}

	magic := string(header[0:4])
	if magic != StreamMagic {
		return 0, 0, fmt.Errorf("invalid magic: %q (expected %q)", magic, StreamMagic)
	}

	version := header[4]
	if version != StreamVersion {
		return 0, 0, fmt.Errorf("unsupported version: %d", version)
	}

	chunkSize := int(binary.LittleEndian.Uint32(header[5:9]))
	if chunkSize <= 0 || chunkSize > 64*1024*1024 {
		return 0, 0, fmt.Errorf("invalid chunk size: %d", chunkSize)
	}

	originalSize := int64(binary.LittleEndian.Uint64(header[9:17]))
	if originalSize < 0 {
		return 0, 0, fmt.Errorf("invalid original size: %d", originalSize)
	}

	return chunkSize, originalSize, nil
}

// counterNonce generates a deterministic nonce from a chunk counter.
// The nonce is the big-endian representation of the counter, zero-padded to nonceSize bytes.
func counterNonce(counter uint64, nonceSize int) []byte {
	nonce := make([]byte, nonceSize)
	// Put the counter in the last 8 bytes of the nonce (big-endian)
	binary.BigEndian.PutUint64(nonce[nonceSize-8:], counter)
	return nonce
}
