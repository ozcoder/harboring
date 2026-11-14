package har

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	bzip2w "github.com/dsnet/compress/bzip2"
)

// maxDecompressedBytes bounds how much a single compressed input may expand to.
// Without this, a small "zip bomb" HAR could decompress to gigabytes and
// exhaust memory. The cap is far above any legitimate HAR while defeating
// decompression bombs. It is a var (rather than a const) so tests can tighten
// it without changing production behaviour.
var maxDecompressedBytes int64 = 1 << 30 // 1 GiB

// ReadAllLimited reads r but refuses to buffer more than maxDecompressedBytes,
// bounding memory even for uncompressed (plaintext) or oversized inputs.
func ReadAllLimited(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxDecompressedBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxDecompressedBytes {
		return nil, fmt.Errorf("input exceeds limit of %d bytes", maxDecompressedBytes)
	}
	return data, nil
}

// ReadFileLimited opens path and reads it through ReadAllLimited.
func ReadFileLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()
	return ReadAllLimited(f)
}

// Decompress inspects the leading magic bytes of data and, if it is a gzip or
// bzip2 stream, returns the decompressed payload. Uncompressed data is returned
// unchanged. This makes reading transparent for both .gz and .bz2 inputs.
func Decompress(data []byte) ([]byte, error) {
	switch {
	case len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b:
		return gunzip(data)
	case len(data) >= 3 && data[0] == 'B' && data[1] == 'Z' && data[2] == 'h':
		return bunzip2(data)
	default:
		return data, nil
	}
}

func gunzip(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer r.Close()
	out, err := io.ReadAll(io.LimitReader(r, maxDecompressedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("gunzip: %w", err)
	}
	if int64(len(out)) > maxDecompressedBytes {
		return nil, fmt.Errorf("decompressed gzip data exceeds limit of %d bytes", maxDecompressedBytes)
	}
	return out, nil
}

// ParseReader reads all bytes from r, transparently decompresses gzip/bzip2/raw
// input, and parses the result as HAR JSON. It is the streaming equivalent of
// ParseFile, enabling pipelines such as `cat foo.har.gz | harboring --scrub`.
func ParseReader(r io.Reader) (*Node, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxDecompressedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	if int64(len(data)) > maxDecompressedBytes {
		return nil, fmt.Errorf("input exceeds limit of %d bytes", maxDecompressedBytes)
	}
	decompressed, err := Decompress(data)
	if err != nil {
		return nil, err
	}
	return ParseBytes(decompressed)
}

func bunzip2(data []byte) ([]byte, error) {
	r := bzip2.NewReader(bytes.NewReader(data))
	out, err := io.ReadAll(io.LimitReader(r, maxDecompressedBytes+1))
	if err != nil {
		return nil, fmt.Errorf("bunzip2: %w", err)
	}
	if int64(len(out)) > maxDecompressedBytes {
		return nil, fmt.Errorf("decompressed bzip2 data exceeds limit of %d bytes", maxDecompressedBytes)
	}
	return out, nil
}

// CompressionFormat identifies an on-disk compression format.
type CompressionFormat string

const (
	FormatNone  CompressionFormat = ""
	FormatGzip  CompressionFormat = "gzip"
	FormatBzip2 CompressionFormat = "bzip2"
)

// FormatForPath chooses a compression format from a file's extension:
// ".gz" -> gzip, ".bz2" -> bzip2, otherwise plain JSON.
func FormatForPath(path string) CompressionFormat {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gz":
		return FormatGzip
	case ".bz2":
		return FormatBzip2
	default:
		return FormatNone
	}
}

// Compress returns data compressed with the given format. FormatNone returns a
// copy of the input unchanged.
func Compress(data []byte, format CompressionFormat) ([]byte, error) {
	switch format {
	case FormatGzip:
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("gzip write: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("gzip close: %w", err)
		}
		return buf.Bytes(), nil
	case FormatBzip2:
		var buf bytes.Buffer
		w, err := bzip2w.NewWriter(&buf, nil)
		if err != nil {
			return nil, fmt.Errorf("bzip2 write: %w", err)
		}
		if _, err := w.Write(data); err != nil {
			return nil, fmt.Errorf("bzip2 write: %w", err)
		}
		if err := w.Close(); err != nil {
			return nil, fmt.Errorf("bzip2 close: %w", err)
		}
		return buf.Bytes(), nil
	case FormatNone:
		out := make([]byte, len(data))
		copy(out, data)
		return out, nil
	default:
		return nil, fmt.Errorf("unknown compression format %q", format)
	}
}

// WriteFileAuto writes data to path, compressing it according to the file's
// extension (.gz -> gzip, .bz2 -> bzip2, otherwise plain JSON).
func WriteFileAuto(path string, data []byte) error {
	out, err := Compress(data, FormatForPath(path))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
