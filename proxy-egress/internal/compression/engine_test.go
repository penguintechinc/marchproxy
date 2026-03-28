package compression_test

import (
	"bytes"
	"testing"
	"time"

	"marchproxy-egress/internal/compression"
)

func TestNewCompressionEngineNotNil(t *testing.T) {
	engine := compression.NewCompressionEngine(nil)
	if engine == nil {
		t.Fatal("NewCompressionEngine(nil) should return non-nil engine")
	}
}

func TestNewCompressionEngineWithConfig(t *testing.T) {
	cfg := &compression.CompressionConfig{
		EnabledAlgorithms: []compression.Algorithm{compression.AlgorithmGzip},
		DefaultAlgorithm:  compression.AlgorithmGzip,
		DefaultLevel:      6,
		MinSize:           100,
		MaxSize:           1024 * 1024,
		ContentTypes:      []string{"text/plain", "application/json"},
		ExcludedTypes:     []string{"image/jpeg"},
		QualityThreshold:  0.8,
		EnableStats:       false,
		StatsInterval:     time.Minute,
	}
	engine := compression.NewCompressionEngine(cfg)
	if engine == nil {
		t.Fatal("NewCompressionEngine with config should return non-nil engine")
	}
}

func TestAlgorithmConstants(t *testing.T) {
	// Verify each constant is a distinct value
	algorithms := []compression.Algorithm{
		compression.AlgorithmGzip,
		compression.AlgorithmDeflate,
		compression.AlgorithmBrotli,
		compression.AlgorithmZstd,
	}
	seen := make(map[compression.Algorithm]bool)
	for _, a := range algorithms {
		if seen[a] {
			t.Errorf("duplicate Algorithm constant: %d", a)
		}
		seen[a] = true
	}
}

func TestGetSupportedAlgorithms(t *testing.T) {
	engine := compression.NewCompressionEngine(nil)
	algos := engine.GetSupportedAlgorithms()
	if algos == nil {
		t.Fatal("GetSupportedAlgorithms should return non-nil slice")
	}
	if len(algos) == 0 {
		t.Error("expected at least one supported algorithm")
	}
}

func TestGetStatsInitial(t *testing.T) {
	cfg := &compression.CompressionConfig{
		EnabledAlgorithms: []compression.Algorithm{compression.AlgorithmGzip},
		DefaultAlgorithm:  compression.AlgorithmGzip,
		MinSize:           0,
		MaxSize:           1024 * 1024 * 10,
		EnableStats:       false,
	}
	engine := compression.NewCompressionEngine(cfg)
	stats := engine.GetStats()
	if stats == nil {
		t.Fatal("GetStats should return non-nil stats")
	}
	if stats.TotalRequests != 0 {
		t.Errorf("expected 0 total requests initially, got %d", stats.TotalRequests)
	}
}

func TestGzipCompressDecompress(t *testing.T) {
	cfg := &compression.CompressionConfig{
		EnabledAlgorithms: []compression.Algorithm{compression.AlgorithmGzip},
		DefaultAlgorithm:  compression.AlgorithmGzip,
		DefaultLevel:      6,
		MinSize:           0, // allow small data
		MaxSize:           1024 * 1024,
		ContentTypes:      []string{"text/plain"},
		ExcludedTypes:     []string{},
		QualityThreshold:  0.99, // high threshold to ensure compression is used
		EnableStats:       false,
	}
	engine := compression.NewCompressionEngine(cfg)

	original := []byte("hello world this is test data for compression testing purposes gzip round-trip")

	req := &compression.CompressionRequest{
		Data:         original,
		ContentType:  "text/plain",
		Algorithm:    compression.AlgorithmGzip,
		Level:        6,
		AcceptHeader: "gzip",
	}

	resp, err := engine.Compress(req)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Compress returned nil response")
	}

	// Decompress and verify
	decompressed, err := engine.Decompress(resp.Data, compression.AlgorithmGzip)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Errorf("decompressed data does not match original\noriginal:     %q\ndecompressed: %q", original, decompressed)
	}
}

func TestDeflateCompressDecompress(t *testing.T) {
	cfg := &compression.CompressionConfig{
		EnabledAlgorithms: []compression.Algorithm{compression.AlgorithmDeflate},
		DefaultAlgorithm:  compression.AlgorithmDeflate,
		DefaultLevel:      6,
		MinSize:           0,
		MaxSize:           1024 * 1024,
		ContentTypes:      []string{"text/plain"},
		ExcludedTypes:     []string{},
		QualityThreshold:  0.99,
		EnableStats:       false,
	}
	engine := compression.NewCompressionEngine(cfg)

	original := []byte("deflate test data for round-trip compression testing with some repetition repetition")

	req := &compression.CompressionRequest{
		Data:         original,
		ContentType:  "text/plain",
		Algorithm:    compression.AlgorithmDeflate,
		Level:        6,
		AcceptHeader: "deflate",
	}

	resp, err := engine.Compress(req)
	if err != nil {
		t.Fatalf("Deflate Compress failed: %v", err)
	}

	decompressed, err := engine.Decompress(resp.Data, compression.AlgorithmDeflate)
	if err != nil {
		t.Fatalf("Deflate Decompress failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Errorf("deflate decompressed data does not match original")
	}
}

func TestDecompressUnsupportedAlgorithm(t *testing.T) {
	engine := compression.NewCompressionEngine(nil)
	// Algorithm value 99 should not be supported
	_, err := engine.Decompress([]byte("fake data"), compression.Algorithm(99))
	if err == nil {
		t.Error("expected error when decompressing with unsupported algorithm")
	}
}

func TestCompressionRequestFields(t *testing.T) {
	req := &compression.CompressionRequest{
		Data:         []byte("test"),
		ContentType:  "text/html",
		Algorithm:    compression.AlgorithmGzip,
		Level:        5,
		AcceptHeader: "gzip, deflate",
	}
	if string(req.Data) != "test" {
		t.Errorf("unexpected Data: %q", req.Data)
	}
	if req.ContentType != "text/html" {
		t.Errorf("unexpected ContentType: %q", req.ContentType)
	}
	if req.Algorithm != compression.AlgorithmGzip {
		t.Errorf("unexpected Algorithm: %d", req.Algorithm)
	}
}

func TestCompressionResponseFields(t *testing.T) {
	resp := &compression.CompressionResponse{
		Data:             []byte("compressed"),
		Algorithm:        compression.AlgorithmGzip,
		Level:            6,
		OriginalSize:     100,
		CompressedSize:   60,
		CompressionRatio: 0.6,
	}
	if resp.OriginalSize != 100 {
		t.Errorf("unexpected OriginalSize: %d", resp.OriginalSize)
	}
	if resp.CompressedSize != 60 {
		t.Errorf("unexpected CompressedSize: %d", resp.CompressedSize)
	}
	if resp.CompressionRatio != 0.6 {
		t.Errorf("unexpected CompressionRatio: %f", resp.CompressionRatio)
	}
}

func TestCompressionSkipsSmallData(t *testing.T) {
	cfg := &compression.CompressionConfig{
		EnabledAlgorithms: []compression.Algorithm{compression.AlgorithmGzip},
		DefaultAlgorithm:  compression.AlgorithmGzip,
		DefaultLevel:      6,
		MinSize:           1000, // data smaller than this is not compressed
		MaxSize:           1024 * 1024,
		ContentTypes:      []string{"text/plain"},
		ExcludedTypes:     []string{},
		QualityThreshold:  0.8,
		EnableStats:       false,
	}
	engine := compression.NewCompressionEngine(cfg)

	// Data smaller than MinSize
	small := []byte("tiny")
	req := &compression.CompressionRequest{
		Data:         small,
		ContentType:  "text/plain",
		AcceptHeader: "gzip",
	}
	resp, err := engine.Compress(req)
	if err != nil {
		t.Fatalf("Compress returned error for small data: %v", err)
	}
	// When compression is skipped, the returned data equals the original
	if !bytes.Equal(resp.Data, small) {
		t.Errorf("expected original data returned when compression skipped")
	}
}

func TestEncoderPoolNotNil(t *testing.T) {
	pool := compression.NewEncoderPool(10)
	if pool == nil {
		t.Fatal("NewEncoderPool should return non-nil pool")
	}
}

func TestEncoderPoolGetPut(t *testing.T) {
	pool := compression.NewEncoderPool(10)

	encoder := pool.GetEncoder(compression.AlgorithmGzip)
	if encoder == nil {
		t.Fatal("GetEncoder(AlgorithmGzip) should return non-nil encoder")
	}

	// Put it back — should not panic
	pool.PutEncoder(compression.AlgorithmGzip, encoder)
}

func TestEncoderPoolUnsupported(t *testing.T) {
	pool := compression.NewEncoderPool(10)
	// Algorithm 99 not in pool
	encoder := pool.GetEncoder(compression.Algorithm(99))
	if encoder != nil {
		t.Error("GetEncoder for unsupported algorithm should return nil")
	}
}
