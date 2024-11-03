package http

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"testing"
)

func Test_Sanity(t *testing.T) {
	data := []byte("test data 123456789012345678901234567890")

	fmt.Println("Original Data:", string(data))
	fmt.Println("Original Data (Hex):", hex.EncodeToString(data))

	// Compress using multiple algorithms
	compressions := []string{"gzip", "deflate", "br", "zstd"}
	dataCompressed, err := compress(data, defaultCompressionFactories, compressions...)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}

	fmt.Println("Compressed Data (Hex):", hex.EncodeToString(dataCompressed))
	fmt.Println("Compressed Data (Base64):", base64.StdEncoding.EncodeToString(dataCompressed))

	// Decompress using the same algorithms
	dataUncompressed, err := decompress(dataCompressed, defaultDecompressionFactories, compressions...)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}

	fmt.Println("Decompressed Data:", string(dataUncompressed))
	fmt.Println("Decompressed Data (Hex):", hex.EncodeToString(dataUncompressed))
}
