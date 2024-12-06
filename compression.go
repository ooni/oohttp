package http

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/lzw"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

type CompressionFactory func(writer io.Writer) (io.Writer, error)
type DecompressionFactory func(reader io.Reader) (io.Reader, error)

type EncodingName = string

type CompressionRegistry map[EncodingName]CompressionFactory
type DecompressionRegistry map[EncodingName]DecompressionFactory

var (
	DefaultCompressionFactories = CompressionRegistry{
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

	DefaultDecompressionFactories = DecompressionRegistry{
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

func compress(data []byte, registry CompressionRegistry, order ...EncodingName) ([]byte, error) {
	var (
		err     error
		writers []io.Writer
		writer  io.Writer
	)

	if registry == nil {
		registry = DefaultCompressionFactories
	}

	dst := bytes.NewBuffer(nil)
	writer = dst

	for idx, compression := range order {
		// fmt.Println(fmt.Sprintf("Compression added: %s", compression))
		mapping, ok := registry[compression]
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

// CompressorWriter compressor writer
type CompressorWriter struct {
	io.Writer

	Registry CompressionRegistry
	Order    []EncodingName

	wrs []io.Writer

	once sync.Once
}

var _ io.WriteCloser = (*CompressorWriter)(nil)

func (cw *CompressorWriter) init() error {
	if cw.Registry == nil {
		cw.Registry = DefaultCompressionFactories
	}
	cw.wrs = nil
	for i := 0; i < len(cw.Order); i++ {
		directive := cw.Order[i]
		directive = strings.Trim(directive, " ")
		if directive == "" {
			continue
		}
		compressorWrapper, exist := cw.Registry[directive]
		if !exist {
			return fmt.Errorf("%s is not supported", directive)
		}
		writer, err := compressorWrapper(cw.Writer)
		if err != nil {
			return fmt.Errorf("compressor wrapper init: %s: %w", directive, err)
		}
		cw.wrs = append(cw.wrs, writer)
		cw.Writer = writer
	}
	return nil
}

// Init initialize decompressor early instead of lazy-initialize on first read op
func (cw *CompressorWriter) Init() (err error) {
	cw.once.Do(func() {
		err = cw.init()
	})
	return
}

// Write write buffer to compressor
func (cw *CompressorWriter) Write(b []byte) (nb int, err error) {
	cw.once.Do(func() {
		err = cw.init()
	})
	if err != nil {
		return
	}
	nb, err = cw.Writer.Write(b)
	return
}

// Close close compressor
func (cw *CompressorWriter) Close() error {
	for i := len(cw.wrs) - 1; i >= 0; i-- {
		if closer, ok := cw.wrs[i].(io.Closer); ok {
			err := closer.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func decompress(data []byte, registry DecompressionRegistry, order ...EncodingName) ([]byte, error) {
	var (
		err     error
		reader  io.Reader
		readers []io.Reader
	)

	if registry == nil {
		registry = DefaultDecompressionFactories
	}

	src := bytes.NewBuffer(data)
	reader = src

	readers = append(readers, src)

	// Reverse the order of compressions for decompression
	for idx := 0; idx < len(order); idx++ {
		compression := order[idx]
		// fmt.Println(fmt.Sprintf("Decompression added: %s", compression))
		mapping, ok := registry[compression]
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

// DecompressorReader decompressor reader
type DecompressorReader struct {
	io.Reader

	Registry DecompressionRegistry
	Order    []EncodingName

	rds []io.Reader

	once sync.Once
}

var _ io.ReadCloser = (*DecompressorReader)(nil)

func (dr *DecompressorReader) init() error {
	if dr.Registry == nil {
		dr.Registry = DefaultDecompressionFactories
	}
	dr.rds = nil
	for i := 0; i < len(dr.Order); i++ {
		directive := dr.Order[i]
		directive = strings.Trim(directive, " ")
		if directive == "" {
			continue
		}
		// fmt.Println(directive)
		decompressorWrapper, exist := dr.Registry[directive]
		if !exist {
			return fmt.Errorf("%s is not supported", directive)
		}
		reader, err := decompressorWrapper(dr.Reader)
		if err != nil {
			return fmt.Errorf("decompressor wrapper init: %s: %w", directive, err)
		}
		dr.rds = append(dr.rds, reader)
		dr.Reader = reader
	}
	return nil
}

// Init initialize decompressor early instead of lazy-initialize on first read op
func (dr *DecompressorReader) Init() (err error) {
	dr.once.Do(func() {
		err = dr.init()
	})
	return
}

// Read read buffer from decompressor
func (dr *DecompressorReader) Read(b []byte) (nb int, err error) {
	dr.once.Do(func() {
		err = dr.init()
	})
	if err != nil {
		return
	}
	nb, err = dr.Reader.Read(b)
	return
}

// Close close decompressor
func (dr *DecompressorReader) Close() error {
	for i := len(dr.rds) - 1; i >= 0; i-- {
		if closer, ok := dr.rds[i].(io.Closer); ok {
			err := closer.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
