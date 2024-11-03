package http

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"errors"
	"fmt"
	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"io"
)

type CompressionFactory func(writer io.Writer) (io.Writer, error)
type DecompressionFactory func(reader io.Reader) (io.Reader, error)

var (
	defaultCompressionFactories = map[string]CompressionFactory{
		"":         func(writer io.Writer) (io.Writer, error) { return writer, nil },
		"identity": func(writer io.Writer) (io.Writer, error) { return writer, nil },
		"gzip":     func(writer io.Writer) (io.Writer, error) { return gzip.NewWriter(writer), nil },
		"zlib":     func(writer io.Writer) (io.Writer, error) { return zlib.NewWriter(writer), nil },
		"br":       func(writer io.Writer) (io.Writer, error) { return brotli.NewWriter(writer), nil },
		"deflate":  func(writer io.Writer) (io.Writer, error) { return flate.NewWriter(writer, -1) },
		// TODO: Confirm compress
		"compress": func(writer io.Writer) (io.Writer, error) { return lzw.NewWriter(writer, lzw.LSB, 8), nil },
		"zstd":     func(writer io.Writer) (io.Writer, error) { return zstd.NewWriter(writer) },
	}

	defaultDecompressionFactories = map[string]DecompressionFactory{
		"":         func(reader io.Reader) (io.Reader, error) { return reader, nil },
		"identity": func(reader io.Reader) (io.Reader, error) { return reader, nil },
		"gzip":     func(reader io.Reader) (io.Reader, error) { return gzip.NewReader(reader) },
		"zlib":     func(reader io.Reader) (io.Reader, error) { return zlib.NewReader(reader) },
		"br":       func(reader io.Reader) (io.Reader, error) { return brotli.NewReader(reader), nil },
		"deflate":  func(reader io.Reader) (io.Reader, error) { return flate.NewReader(reader), nil },
		"compress": func(reader io.Reader) (io.Reader, error) { return lzw.NewReader(reader, lzw.LSB, 8), nil },
		"zstd":     func(reader io.Reader) (io.Reader, error) { return zstd.NewReader(reader) },
	}
)

func compress(data []byte, compressions map[string]CompressionFactory, compressionOrder ...string) ([]byte, error) {
	var (
		err     error
		writers []io.Writer
		writer  io.Writer
	)

	if compressions == nil {
		compressions = defaultCompressionFactories
	}

	dst := bytes.NewBuffer(nil)
	writer = dst

	for idx, compression := range compressionOrder {
		// fmt.Println(fmt.Sprintf("Compression added: %s", compression))
		mapping, ok := compressions[compression]
		if !ok {
			return nil, errors.New(compression + " is not supported")
		}
		writer, err = mapping(writer)
		if err != nil {
			return nil, fmt.Errorf("mapping[%d:%s]: %w", idx, compression, err)
		}

		writers = append(writers, writer)
	}

	_, err = writers[len(writers)-1].Write(data)
	if err != nil {
		return nil, fmt.Errorf("writer.Write: %w", err)
	}

	// Close all writers in reverse order to ensure all data is flushed
	for i := len(writers) - 1; i >= 0; i-- {
		err = writers[i].(io.Closer).Close()
		if err != nil {
			return nil, fmt.Errorf("writers[%d].(io.Closer).Close: %w", i, err)
		}
	}

	// fmt.Printf("lenIn: %d lenOut: %d\n", len(data), dst.Len())
	return dst.Bytes(), nil
}

func decompress(data []byte, compressions map[string]DecompressionFactory, compressionOrder ...string) ([]byte, error) {
	var (
		err     error
		reader  io.Reader
		readers []io.Reader
	)

	if compressions == nil {
		compressions = defaultDecompressionFactories
	}

	src := bytes.NewBuffer(data)
	reader = src

	readers = append(readers, src)

	// Reverse the order of compressions for decompression
	for idx := 0; idx < len(compressionOrder); idx++ {
		compression := compressionOrder[idx]
		// fmt.Println(fmt.Sprintf("Decompression added: %s", compression))
		mapping, ok := compressions[compression]
		if !ok {
			return nil, errors.New(compression + " is not supported")
		}
		reader, err = mapping(reader)
		if err != nil {
			return nil, fmt.Errorf("mapping[%d:%s]: %w", idx, compression, err)
		}

		readers = append(readers, reader)
	}

	dataOut, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("io.ReadAll: %w", err)
	}

	for _, readerObj := range readers {
		typedReader, ok := readerObj.(io.Closer)
		if ok {
			defer typedReader.Close()
		}
	}

	// fmt.Printf("lenIn: %d lenOut: %d\n", len(data), len(dataOut))
	return dataOut, nil
}

func decompressReader(src io.Reader, compressions map[string]DecompressionFactory, compressionOrder []string) (io.ReadCloser, error) {
	var (
		err error
	)

	if compressions == nil {
		compressions = defaultDecompressionFactories
	}

	result := &bodyDecompressorReader{
		reader:           src,
		CompressionOrder: compressionOrder,
	}

	result.readers = append(result.readers, result.reader)

	// Reverse the order of compressions for decompression
	for idx := 0; idx < len(compressionOrder); idx++ {
		compression := compressionOrder[idx]
		// fmt.Println(fmt.Sprintf("Decompression added: %s", compression))
		mapping, ok := compressions[compression]
		if !ok {
			return nil, errors.New(compression + " is not supported")
		}
		result.reader, err = mapping(result.reader)
		if err != nil {
			return nil, fmt.Errorf("mapping[%d:%s]: %w", idx, compression, err)
		}

		result.readers = append(result.readers, result.reader)
	}

	return result, nil
}

type bodyDecompressorReader struct {
	reader           io.Reader
	readers          []io.Reader
	Factory          map[string]DecompressionFactory
	CompressionOrder []string
}

func (body *bodyDecompressorReader) Read(p []byte) (n int, err error) {
	return body.reader.Read(p)
}

func (body *bodyDecompressorReader) Close() error {
	for _, readerObj := range body.readers {
		typedReader, ok := readerObj.(io.Closer)
		if ok {
			err := typedReader.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
