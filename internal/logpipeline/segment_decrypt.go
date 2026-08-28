package logpipeline

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"ad-event-processor/pkg/logger"

	"github.com/klauspost/compress/zstd"
)

type decryptedSegmentReader struct {
	file    *os.File
	aesgcm  cipher.AEAD
	decoder *zstd.Decoder
	buf     []byte
	off     int
	done    bool
	header  [4]byte
	nonce   [12]byte
}

func openPlaintextSegment(path string, decryptKey []byte) (io.ReadCloser, error) {
	if strings.HasSuffix(path, readySuffix) {
		if len(decryptKey) == 0 {
			passphrase := os.Getenv("LOG_ENCRYPTION_KEY")
			if passphrase == "" {
				passphrase = "default-ad-event-processor-logger-fallback-passphrase-change-me"
			}
			decryptKey = logger.DeriveKey(passphrase)
		}
		return openDecryptedSegment(path, decryptKey)
	}
	return os.Open(path)
}

func openDecryptedSegment(path string, key []byte) (*decryptedSegmentReader, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return &decryptedSegmentReader{
		file:    file,
		aesgcm:  aesgcm,
		decoder: decoder,
	}, nil
}

func (sr *decryptedSegmentReader) Read(p []byte) (int, error) {
	for {
		if sr.off < len(sr.buf) {
			n := copy(p, sr.buf[sr.off:])
			sr.off += n
			return n, nil
		}
		if sr.done {
			return 0, io.EOF
		}
		if err := sr.readNextBlock(); err != nil {
			if errors.Is(err, io.EOF) {
				sr.done = true
				return 0, io.EOF
			}
			return 0, err
		}
		sr.off = 0
	}
}

func (sr *decryptedSegmentReader) Close() error {
	if sr.decoder != nil {
		sr.decoder.Close()
	}
	if sr.file != nil {
		return sr.file.Close()
	}
	return nil
}

func (sr *decryptedSegmentReader) readNextBlock() error {
	_, err := io.ReadFull(sr.file, sr.header[:])
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return io.EOF
	}
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(sr.header[:])
	if length < 12+16 {
		return fmt.Errorf("invalid encrypted block length: %d", length)
	}

	if _, err := io.ReadFull(sr.file, sr.nonce[:]); err != nil {
		return err
	}

	ciphertextLen := length - 12
	ciphertext := make([]byte, ciphertextLen)
	if _, err := io.ReadFull(sr.file, ciphertext); err != nil {
		return err
	}

	plaintext, err := sr.aesgcm.Open(nil, sr.nonce[:], ciphertext, nil)
	if err != nil {
		return err
	}

	decompressed, err := sr.decoder.DecodeAll(plaintext, sr.buf[:0])
	if err != nil {
		return fmt.Errorf("decompress encrypted block: %w", err)
	}
	sr.buf = decompressed
	return nil
}
