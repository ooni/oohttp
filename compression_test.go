package http

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
)

func TestCompressionDecompressionRoundTrip(t *testing.T) {
	type tc struct {
		input              []byte
		compressions       []string
		expectedCompressed []byte
		expected           []byte
	}

	for i, tc := range []tc{
		{
			input:        []byte("test data 123456789012345678901234567890"),
			compressions: []string{"gzip", "deflate", "br", "zstd"},
			expectedCompressed: (func() []byte {
				dec, err := hex.DecodeString("1f8b08000000000000ff7a249679e1c6aaffd35b1f6a323773277b6615763a7925792e6a5a387163e7ca993b5b97322677086c5e30e5f86173f3998c8c0c0cffff03020000ffff66bd043d34000000")
				if err != nil {
					panic(err)
				}
				return dec
			})(),
			expected: []byte("test data 123456789012345678901234567890"),
		},
		{
			input:              []byte("foo bar baz\n\n"),
			compressions:       []string{},
			expectedCompressed: []byte("foo bar baz\n\n"),
			expected:           []byte("foo bar baz\n\n"),
		},
		{
			input:              []byte("hello"),
			compressions:       []string{""},
			expectedCompressed: []byte("hello"),
			expected:           []byte("hello"),
		},
		{
			input:              []byte("hello"),
			compressions:       []string{"identity"},
			expectedCompressed: []byte("hello"),
			expected:           []byte("hello"),
		},
		{
			input:        []byte("hello"),
			compressions: []string{"gzip"},
			expectedCompressed: []byte{
				0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff,
				0xca, 0x48, 0xcd, 0xc9, 0xc9, 0x7, 0x4, 0x0, 0x0, 0xff, 0xff,
				0x86, 0xa6, 0x10, 0x36, 0x5, 0x0, 0x0, 0x0,
			},
			expected: []byte("hello"),
		},
		{
			input:        []byte("foo bar baz\n\n"),
			compressions: []string{"gzip", "br"},
			expectedCompressed: []byte{
				0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff,
				0xe2, 0x66, 0x6b, 0x48, 0xcb, 0xcf, 0x57, 0x48, 0x4a,
				0x2c, 0x52, 0x48, 0x4a, 0xac, 0xe2, 0xe2, 0x62, 0x6, 0x4,
				0x0, 0x0, 0xff, 0xff, 0xcb, 0xa9, 0xea, 0xd4, 0x11, 0x0, 0x0, 0x0,
			},
			expected: []byte("foo bar baz\n\n"),
		},
		{
			input:        []byte("hello"),
			compressions: []string{"gzip", "deflate", "br", "zstd"},
			expectedCompressed: []byte{
				0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x7a, 0xf5,
				0x2c, 0xe3, 0xc2, 0x8d, 0x55, 0xff, 0xa7, 0xb7, 0x3a, 0x4e, 0x6e,
				0x54, 0x54, 0x36, 0x55, 0x57, 0xaf, 0x2f, 0x7c, 0xf7, 0x87, 0x2f, 0xcd, 0x81,
				0x81, 0x81, 0xe1, 0xff, 0x7f, 0x40, 0x0, 0x0, 0x0, 0xff, 0xff, 0x51, 0x13, 0xb7,
				0x9e, 0x1d, 0x0, 0x0, 0x0,
			},
			expected: []byte("hello"),
		},
	} {
		buf := bytes.NewBuffer([]byte{})

		writer := &CompressorWriter{
			Writer: buf,
			Order:  tc.compressions,
		}

		(func() {
			var writer io.Writer = writer
			if closer, ok := writer.(io.Closer); ok {
				defer closer.Close()
			}

			nb, err := writer.Write(tc.input)
			if err != nil {
				t.Errorf("compressor write: %s", err)
			}
			fmt.Println("compressor write nb", nb)
		})()

		// peek buffer
		if !bytes.Equal(tc.expectedCompressed, buf.Bytes()) {
			t.Errorf("unexpected compression result: %d: %+#v", i, tc.compressions)
		}
		fmt.Printf("raw buf %+#v\n", buf.Bytes())

		reader := &DecompressorReader{
			Reader: bytes.NewReader(buf.Bytes()),
			Order:  tc.compressions,
		}

		(func() {
			var reader io.Reader = reader
			if closer, ok := reader.(io.Closer); ok {
				defer closer.Close()
			}

			actual, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("decompressor read: %s", err)
			}
			if !bytes.Equal(actual, tc.expected) {
				t.Errorf("result mismatch: %d: %+#v", i, tc.compressions)
			}
		})()
	}
}

func Test_Sanity(t *testing.T) {
	data := []byte("test data 123456789012345678901234567890")

	fmt.Println("Original Data:", string(data))
	fmt.Println("Original Data (Hex):", hex.EncodeToString(data))

	// Compress using multiple algorithms
	compressions := []string{"gzip", "deflate", "br", "zstd"}
	dataCompressed, err := compress(data, DefaultCompressionFactories, compressions...)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}

	fmt.Println("Compressed Data (Hex):", hex.EncodeToString(dataCompressed))
	fmt.Println("Compressed Data (Base64):", base64.StdEncoding.EncodeToString(dataCompressed))

	// Decompress using the same algorithms
	dataUncompressed, err := decompress(dataCompressed, DefaultDecompressionFactories, compressions...)
	if err != nil {
		t.Fatalf("decompress: %v", err)
	}

	fmt.Println("Decompressed Data:", string(dataUncompressed))
	fmt.Println("Decompressed Data (Hex):", hex.EncodeToString(dataUncompressed))
}
