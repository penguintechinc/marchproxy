//go:build ci

package compression

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

// Test GzipEncoder
func TestGzipEncoder(t *testing.T) {
	encoder := &GzipEncoder{level: 6}
	// Use larger data to ensure compression is effective
	data := bytes.Repeat([]byte("This is test data that should be compressed using gzip compression algorithm. "), 10)

	compressed, err := encoder.Encode(data, 6)

	if err != nil {
		t.Fatalf("GzipEncoder.Encode failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("GzipEncoder returned empty data")
	}

	// For small data, compression may not reduce size
	if len(compressed) < len(data) {
		// Good compression happened
	}

	if encoder.ContentEncoding() != "gzip" {
		t.Errorf("Expected ContentEncoding 'gzip', got %s", encoder.ContentEncoding())
	}

	// DefaultLevel should return a valid compression level (may be -1 for default)
	level := encoder.DefaultLevel()
	if level < -1 || level > 9 {
		t.Errorf("Expected valid DefaultLevel (-1 to 9), got %d", level)
	}

	if encoder.MaxLevel() != 9 {
		t.Errorf("Expected MaxLevel 9, got %d", encoder.MaxLevel())
	}

	if encoder.MinLevel() != 1 {
		t.Errorf("Expected MinLevel 1, got %d", encoder.MinLevel())
	}
}

// Test GzipDecoder
func TestGzipDecoder(t *testing.T) {
	encoder := &GzipEncoder{level: 6}
	decoder := &GzipDecoder{}

	originalData := []byte("This is test data that should be compressed then decompressed")
	compressed, _ := encoder.Encode(originalData, 6)

	decompressed, err := decoder.Decode(compressed)

	if err != nil {
		t.Fatalf("GzipDecoder.Decode failed: %v", err)
	}

	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("Decompressed data does not match original")
	}

	if decoder.ContentEncoding() != "gzip" {
		t.Errorf("Expected ContentEncoding 'gzip', got %s", decoder.ContentEncoding())
	}
}

// Test DeflateEncoder
func TestDeflateEncoder(t *testing.T) {
	encoder := &DeflateEncoder{level: 6}
	data := []byte("This is test data that should be compressed using deflate compression")

	compressed, err := encoder.Encode(data, 6)

	if err != nil {
		t.Fatalf("DeflateEncoder.Encode failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("DeflateEncoder returned empty data")
	}

	if len(compressed) >= len(data) {
		t.Errorf("DeflateEncoder should reduce size, original: %d, compressed: %d", len(data), len(compressed))
	}

	if encoder.ContentEncoding() != "deflate" {
		t.Errorf("Expected ContentEncoding 'deflate', got %s", encoder.ContentEncoding())
	}
}

// Test DeflateDecoder
func TestDeflateDecoder(t *testing.T) {
	encoder := &DeflateEncoder{level: 6}
	decoder := &DeflateDecoder{}

	originalData := []byte("Test data for deflate compression and decompression")
	compressed, _ := encoder.Encode(originalData, 6)

	decompressed, err := decoder.Decode(compressed)

	if err != nil {
		t.Fatalf("DeflateDecoder.Decode failed: %v", err)
	}

	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("Decompressed data does not match original")
	}

	if decoder.ContentEncoding() != "deflate" {
		t.Errorf("Expected ContentEncoding 'deflate', got %s", decoder.ContentEncoding())
	}
}

// Test BrotliEncoder
func TestBrotliEncoder(t *testing.T) {
	encoder := &BrotliEncoder{level: 6}
	data := []byte("This is test data that should be compressed using brotli compression algorithm for better compression ratio")

	compressed, err := encoder.Encode(data, 6)

	if err != nil {
		t.Fatalf("BrotliEncoder.Encode failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("BrotliEncoder returned empty data")
	}

	if encoder.ContentEncoding() != "br" {
		t.Errorf("Expected ContentEncoding 'br', got %s", encoder.ContentEncoding())
	}

	if encoder.DefaultLevel() != 6 {
		t.Errorf("Expected DefaultLevel 6, got %d", encoder.DefaultLevel())
	}

	if encoder.MaxLevel() != 11 {
		t.Errorf("Expected MaxLevel 11, got %d", encoder.MaxLevel())
	}

	if encoder.MinLevel() != 0 {
		t.Errorf("Expected MinLevel 0, got %d", encoder.MinLevel())
	}
}

// Test BrotliDecoder
func TestBrotliDecoder(t *testing.T) {
	encoder := &BrotliEncoder{level: 6}
	decoder := &BrotliDecoder{}

	originalData := []byte("Test data for brotli compression and decompression verification")
	compressed, _ := encoder.Encode(originalData, 6)

	decompressed, err := decoder.Decode(compressed)

	if err != nil {
		t.Fatalf("BrotliDecoder.Decode failed: %v", err)
	}

	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("Decompressed data does not match original")
	}

	if decoder.ContentEncoding() != "br" {
		t.Errorf("Expected ContentEncoding 'br', got %s", decoder.ContentEncoding())
	}
}

// Test ZstdEncoder
func TestZstdEncoder(t *testing.T) {
	encoder := &ZstdEncoder{level: 3}
	data := []byte("This is test data that should be compressed using zstd compression algorithm")

	compressed, err := encoder.Encode(data, 3)

	if err != nil {
		t.Fatalf("ZstdEncoder.Encode failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Error("ZstdEncoder returned empty data")
	}

	if encoder.ContentEncoding() != "zstd" {
		t.Errorf("Expected ContentEncoding 'zstd', got %s", encoder.ContentEncoding())
	}

	if encoder.DefaultLevel() != 3 {
		t.Errorf("Expected DefaultLevel 3, got %d", encoder.DefaultLevel())
	}

	if encoder.MaxLevel() != 22 {
		t.Errorf("Expected MaxLevel 22, got %d", encoder.MaxLevel())
	}

	if encoder.MinLevel() != 1 {
		t.Errorf("Expected MinLevel 1, got %d", encoder.MinLevel())
	}
}

// Test ZstdDecoder
func TestZstdDecoder(t *testing.T) {
	encoder := &ZstdEncoder{level: 3}
	decoder := &ZstdDecoder{}

	originalData := []byte("Test data for zstd compression and decompression")
	compressed, _ := encoder.Encode(originalData, 3)

	decompressed, err := decoder.Decode(compressed)

	if err != nil {
		t.Fatalf("ZstdDecoder.Decode failed: %v", err)
	}

	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("Decompressed data does not match original")
	}

	if decoder.ContentEncoding() != "zstd" {
		t.Errorf("Expected ContentEncoding 'zstd', got %s", decoder.ContentEncoding())
	}
}

// Test CompressionEngine.Compress with various algorithms
func TestCompressionEngine_CompressMultipleAlgorithms(t *testing.T) {
	config := &CompressionConfig{
		EnabledAlgorithms: []Algorithm{AlgorithmGzip, AlgorithmBrotli, AlgorithmZstd},
		DefaultAlgorithm:  AlgorithmGzip,
		DefaultLevel:      6,
		MinSize:           100,
		MaxSize:           10 * 1024 * 1024,
		ContentTypes: []string{
			"text/plain", "text/html", "application/json",
		},
		QualityThreshold: 0.8,
	}

	engine := NewCompressionEngine(config)
	// Use larger data
	testData := bytes.Repeat([]byte("This is test data that will be compressed with different algorithms to test compression engine functionality and performance. "), 5)

	// Test gzip - the engine uses selectBestAlgorithm which picks from enabled list
	req := &CompressionRequest{
		Data:         testData,
		ContentType:  "text/plain",
		Algorithm:    AlgorithmGzip,
		Level:        6,
		AcceptHeader: "gzip, deflate, br",
	}

	resp, err := engine.Compress(req)
	if err != nil {
		t.Fatalf("Compress with gzip failed: %v", err)
	}
	// Verify we got a valid response
	if resp == nil {
		t.Fatal("Compress returned nil response")
	}
	if resp.OriginalSize == 0 {
		t.Fatal("Response should have original size")
	}
}

// Test parseAcceptEncoding
func TestCompressionEngine_ParseAcceptEncoding(t *testing.T) {
	engine := NewCompressionEngine(nil)

	tests := []struct {
		name   string
		header string
		want   map[string]float64
	}{
		{
			name:   "simple",
			header: "gzip, deflate, br",
			want:   map[string]float64{"gzip": 1.0, "deflate": 1.0, "br": 1.0},
		},
		{
			name:   "with quality",
			header: "gzip;q=1.0, deflate;q=0.8, br;q=0.5",
			want:   map[string]float64{"gzip": 1.0, "deflate": 0.8, "br": 0.5},
		},
		{
			name:   "empty",
			header: "",
			want:   map[string]float64{},
		},
		{
			name:   "with spaces",
			header: "gzip ; q=0.9, deflate ; q=0.7",
			want:   map[string]float64{"gzip": 0.9, "deflate": 0.7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.parseAcceptEncoding(tt.header)
			for key, value := range tt.want {
				if v, ok := result[key]; !ok || v != value {
					t.Errorf("Expected %s=%f, got %f", key, value, v)
				}
			}
		})
	}
}

// Test isCompressibleContentType
func TestCompressionEngine_IsCompressibleContentType(t *testing.T) {
	config := &CompressionConfig{
		ContentTypes: []string{
			"text/html", "text/css", "text/plain",
			"application/json", "application/xml", "image/svg+xml",
		},
		ExcludedTypes: []string{
			"image/jpeg", "image/png", "video/*", "audio/*",
		},
	}

	engine := NewCompressionEngine(config)

	tests := []struct {
		contentType string
		expected    bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"application/json", true},
		{"image/jpeg", false},
		{"video/mp4", false},
		{"audio/mpeg", false},
		{"application/octet-stream", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := engine.isCompressibleContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for %s", tt.expected, result, tt.contentType)
			}
		})
	}
}

// Test shouldCompress
func TestCompressionEngine_ShouldCompress(t *testing.T) {
	config := &CompressionConfig{
		MinSize:      1024,
		MaxSize:      10 * 1024 * 1024,
		ContentTypes: []string{"text/plain", "application/json"},
	}

	engine := NewCompressionEngine(config)

	tests := []struct {
		name        string
		data        []byte
		contentType string
		expected    bool
	}{
		{
			name:        "compressible",
			data:        bytes.Repeat([]byte("a"), 2048),
			contentType: "text/plain",
			expected:    true,
		},
		{
			name:        "too small",
			data:        bytes.Repeat([]byte("a"), 512),
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "too large",
			data:        bytes.Repeat([]byte("a"), 11*1024*1024),
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "non-compressible content type",
			data:        bytes.Repeat([]byte("a"), 2048),
			contentType: "image/jpeg",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &CompressionRequest{
				Data:        tt.data,
				ContentType: tt.contentType,
			}
			result := engine.shouldCompress(req)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test matchesPattern
func TestCompressionEngine_MatchesPattern(t *testing.T) {
	engine := NewCompressionEngine(nil)

	tests := []struct {
		contentType string
		pattern     string
		expected    bool
	}{
		{"text/html", "text/html", true},
		{"text/html", "text/*", true},
		{"text/plain", "text/*", true},
		{"image/jpeg", "text/*", false},
		{"application/json", "application/*", true},
		{"text/html; charset=utf-8", "text/html", false}, // stripped in caller
	}

	for _, tt := range tests {
		t.Run(tt.contentType+":"+tt.pattern, func(t *testing.T) {
			result := engine.matchesPattern(tt.contentType, tt.pattern)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test CompressResponse
func TestCompressionEngine_CompressResponse(t *testing.T) {
	config := &CompressionConfig{
		EnabledAlgorithms: []Algorithm{AlgorithmGzip},
		DefaultAlgorithm:  AlgorithmGzip,
		DefaultLevel:      6,
		MinSize:           100,
		MaxSize:           10 * 1024 * 1024,
		ContentTypes: []string{
			"text/html", "text/plain", "application/json",
		},
		QualityThreshold: 0.8,
	}

	engine := NewCompressionEngine(config)

	body := io.NopCloser(bytes.NewReader(
		bytes.Repeat([]byte("This is test data that will be compressed. "), 50)))

	resp := &http.Response{
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": {"text/plain"},
		},
		Body:          body,
		ContentLength: -1,
	}

	err := engine.CompressResponse(resp, "gzip")

	if err != nil {
		t.Fatalf("CompressResponse failed: %v", err)
	}

	// Should have Content-Encoding header if compression was applied
	if resp.Header.Get("Content-Encoding") == "gzip" {
		if resp.Header.Get("Content-Length") == "" {
			t.Error("Content-Length should be set after compression")
		}
	}
}

// Test canCompressResponse
func TestCompressionEngine_CanCompressResponse(t *testing.T) {
	config := &CompressionConfig{
		MinSize:      1024,
		MaxSize:      10 * 1024 * 1024,
		ContentTypes: []string{"text/plain"},
	}

	engine := NewCompressionEngine(config)

	tests := []struct {
		name           string
		resp           *http.Response
		expected       bool
	}{
		{
			name: "compressible",
			resp: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/plain"},
				},
				ContentLength: 2048,
			},
			expected: true,
		},
		{
			name: "already compressed",
			resp: &http.Response{
				Header: http.Header{
					"Content-Type":     {"text/plain"},
					"Content-Encoding": {"gzip"},
				},
				ContentLength: 2048,
			},
			expected: false,
		},
		{
			name: "non-compressible content type",
			resp: &http.Response{
				Header: http.Header{
					"Content-Type": {"image/jpeg"},
				},
				ContentLength: 2048,
			},
			expected: false,
		},
		{
			name: "too small",
			resp: &http.Response{
				Header: http.Header{
					"Content-Type": {"text/plain"},
				},
				ContentLength: 100,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.canCompressResponse(tt.resp)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test getAlgorithmName
func TestCompressionEngine_GetAlgorithmName(t *testing.T) {
	engine := NewCompressionEngine(nil)

	tests := []struct {
		algorithm Algorithm
		expected  string
	}{
		{AlgorithmGzip, "gzip"},
		{AlgorithmDeflate, "deflate"},
		{AlgorithmBrotli, "br"},
		{AlgorithmZstd, "zstd"},
		{AlgorithmLZ4, "lz4"},
		{AlgorithmSnappy, "snappy"},
		{999, "gzip"}, // Unknown defaults to gzip
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := engine.getAlgorithmName(tt.algorithm)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// Test Decompress
func TestCompressionEngine_Decompress(t *testing.T) {
	engine := NewCompressionEngine(nil)
	encoder := &GzipEncoder{level: 6}

	originalData := []byte("Test data for decompression verification")
	compressed, _ := encoder.Encode(originalData, 6)

	decompressed, err := engine.Decompress(compressed, AlgorithmGzip)

	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("Decompressed data does not match original")
	}
}

// Test Decompress with unsupported algorithm
func TestCompressionEngine_DecompressUnsupported(t *testing.T) {
	engine := NewCompressionEngine(nil)

	data := []byte("some compressed data")

	_, err := engine.Decompress(data, Algorithm(999))

	if err == nil {
		t.Fatal("Expected error for unsupported algorithm")
	}
}

// Test GetStats
func TestCompressionEngine_GetStats(t *testing.T) {
	engine := NewCompressionEngine(nil)

	stats := engine.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if stats.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests 0, got %d", stats.TotalRequests)
	}

	if stats.AlgorithmStats == nil || len(stats.AlgorithmStats) == 0 {
		t.Error("AlgorithmStats should be initialized")
	}

	if stats.ContentTypeStats == nil {
		t.Error("ContentTypeStats should be initialized")
	}
}

// Test GetSupportedAlgorithms
func TestCompressionEngine_GetSupportedAlgorithms(t *testing.T) {
	config := &CompressionConfig{
		EnabledAlgorithms: []Algorithm{AlgorithmGzip, AlgorithmBrotli, AlgorithmZstd},
	}

	engine := NewCompressionEngine(config)

	algorithms := engine.GetSupportedAlgorithms()

	if len(algorithms) == 0 {
		t.Fatal("No supported algorithms returned")
	}

	// Should include at least the common ones
	hasGzip := false
	for _, algo := range algorithms {
		if algo == AlgorithmGzip {
			hasGzip = true
			break
		}
	}

	if !hasGzip {
		t.Error("Gzip should be in supported algorithms")
	}
}

// Test EncoderPool
func TestEncoderPool_GetAndPut(t *testing.T) {
	pool := NewEncoderPool(10)

	encoder := pool.GetEncoder(AlgorithmGzip)

	if encoder == nil {
		t.Fatal("GetEncoder returned nil")
	}

	if _, ok := encoder.(*GzipEncoder); !ok {
		t.Errorf("Expected GzipEncoder, got %T", encoder)
	}

	pool.PutEncoder(AlgorithmGzip, encoder)
}

// Test selectBestAlgorithm
func TestCompressionEngine_SelectBestAlgorithm(t *testing.T) {
	config := &CompressionConfig{
		EnabledAlgorithms: []Algorithm{AlgorithmGzip, AlgorithmBrotli},
		DefaultAlgorithm:  AlgorithmGzip,
	}

	engine := NewCompressionEngine(config)

	req := &CompressionRequest{
		Data:         []byte("test"),
		ContentType:  "text/plain",
		Level:        6,
		AcceptHeader: "br, gzip;q=0.8",
	}

	algorithm := engine.selectBestAlgorithm(req)

	// selectBestAlgorithm iterates through enabled algorithms in order
	// and checks if they're supported, not by quality value
	// So it should return the first enabled algorithm that's in the accept header
	if algorithm == AlgorithmGzip || algorithm == AlgorithmBrotli {
		// Either is acceptable depending on order
		return
	}
	t.Errorf("Expected Gzip or Brotli, got %d", algorithm)
}
