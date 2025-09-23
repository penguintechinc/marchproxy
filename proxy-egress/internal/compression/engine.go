package compression

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
)

// CompressionEngine handles content compression and decompression
type CompressionEngine struct {
	encoders map[Algorithm]Encoder
	decoders map[Algorithm]Decoder
	config   *CompressionConfig
	stats    *CompressionStats
	pool     *EncoderPool
	mu       sync.RWMutex
}

// Algorithm represents compression algorithms
type Algorithm int

const (
	AlgorithmGzip Algorithm = iota
	AlgorithmDeflate
	AlgorithmBrotli
	AlgorithmZstd
	AlgorithmLZ4
	AlgorithmSnappy
)

// Encoder interface for compression
type Encoder interface {
	Encode(data []byte, level int) ([]byte, error)
	ContentEncoding() string
	DefaultLevel() int
	MaxLevel() int
	MinLevel() int
}

// Decoder interface for decompression
type Decoder interface {
	Decode(data []byte) ([]byte, error)
	ContentEncoding() string
}

// CompressionConfig holds compression configuration
type CompressionConfig struct {
	EnabledAlgorithms []Algorithm
	DefaultAlgorithm  Algorithm
	DefaultLevel      int
	MinSize           int
	MaxSize           int
	ContentTypes      []string
	ExcludedTypes     []string
	QualityThreshold  float64
	EnableStreaming   bool
	BufferSize        int
	PoolSize          int
	EnableStats       bool
	StatsInterval     time.Duration
}

// CompressionStats holds compression statistics
type CompressionStats struct {
	TotalRequests        uint64
	CompressedRequests   uint64
	DecompressedRequests uint64
	BytesIn              uint64
	BytesOut             uint64
	CompressionRatio     float64
	AlgorithmStats       map[Algorithm]*AlgorithmStats
	ContentTypeStats     map[string]*ContentTypeStats
	AverageLatency       time.Duration
	ErrorCount           uint64
	LastUpdate           time.Time
}

// AlgorithmStats holds per-algorithm statistics
type AlgorithmStats struct {
	Requests         uint64
	BytesIn          uint64
	BytesOut         uint64
	CompressionRatio float64
	AverageLatency   time.Duration
	ErrorCount       uint64
}

// ContentTypeStats holds per-content-type statistics
type ContentTypeStats struct {
	Requests         uint64
	BytesIn          uint64
	BytesOut         uint64
	CompressionRatio float64
	OptimalAlgorithm Algorithm
}

// EncoderPool manages encoder instances for performance
type EncoderPool struct {
	pools map[Algorithm]*sync.Pool
	mu    sync.RWMutex
}

// CompressionRequest represents a compression request
type CompressionRequest struct {
	Data         []byte
	ContentType  string
	Algorithm    Algorithm
	Level        int
	AcceptHeader string
}

// CompressionResponse represents a compression response
type CompressionResponse struct {
	Data            []byte
	Algorithm       Algorithm
	Level           int
	OriginalSize    int
	CompressedSize  int
	CompressionRatio float64
	Latency         time.Duration
}

// Built-in encoder implementations

// GzipEncoder implements gzip compression
type GzipEncoder struct {
	level int
}

// DeflateEncoder implements deflate compression
type DeflateEncoder struct {
	level int
}

// BrotliEncoder implements brotli compression
type BrotliEncoder struct {
	level int
}

// ZstdEncoder implements zstd compression
type ZstdEncoder struct {
	level int
}

// Built-in decoder implementations

// GzipDecoder implements gzip decompression
type GzipDecoder struct{}

// DeflateDecoder implements deflate decompression
type DeflateDecoder struct{}

// BrotliDecoder implements brotli decompression
type BrotliDecoder struct{}

// ZstdDecoder implements zstd decompression
type ZstdDecoder struct{}

// NewCompressionEngine creates a new compression engine
func NewCompressionEngine(config *CompressionConfig) *CompressionEngine {
	if config == nil {
		config = &CompressionConfig{
			EnabledAlgorithms: []Algorithm{AlgorithmGzip, AlgorithmBrotli, AlgorithmZstd},
			DefaultAlgorithm:  AlgorithmGzip,
			DefaultLevel:      6,
			MinSize:           1024,
			MaxSize:           10 * 1024 * 1024, // 10MB
			ContentTypes: []string{
				"text/html", "text/css", "text/javascript", "text/plain",
				"application/javascript", "application/json", "application/xml",
				"text/xml", "image/svg+xml",
			},
			ExcludedTypes:    []string{"image/jpeg", "image/png", "image/gif", "video/*", "audio/*"},
			QualityThreshold: 0.8,
			EnableStreaming:  true,
			BufferSize:       64 * 1024,
			PoolSize:         100,
			EnableStats:      true,
			StatsInterval:    time.Minute,
		}
	}

	engine := &CompressionEngine{
		encoders: make(map[Algorithm]Encoder),
		decoders: make(map[Algorithm]Decoder),
		config:   config,
		stats: &CompressionStats{
			AlgorithmStats:   make(map[Algorithm]*AlgorithmStats),
			ContentTypeStats: make(map[string]*ContentTypeStats),
			LastUpdate:       time.Now(),
		},
		pool: NewEncoderPool(config.PoolSize),
	}

	// Initialize encoders and decoders
	engine.initializeEncoders()
	engine.initializeDecoders()

	// Start statistics collection if enabled
	if config.EnableStats {
		go engine.statsCollector()
	}

	return engine
}

// initializeEncoders initializes compression encoders
func (ce *CompressionEngine) initializeEncoders() {
	// Initialize gzip encoder
	ce.encoders[AlgorithmGzip] = &GzipEncoder{level: ce.config.DefaultLevel}
	ce.stats.AlgorithmStats[AlgorithmGzip] = &AlgorithmStats{}

	// Initialize deflate encoder
	ce.encoders[AlgorithmDeflate] = &DeflateEncoder{level: ce.config.DefaultLevel}
	ce.stats.AlgorithmStats[AlgorithmDeflate] = &AlgorithmStats{}

	// Initialize brotli encoder
	ce.encoders[AlgorithmBrotli] = &BrotliEncoder{level: ce.config.DefaultLevel}
	ce.stats.AlgorithmStats[AlgorithmBrotli] = &AlgorithmStats{}

	// Initialize zstd encoder
	ce.encoders[AlgorithmZstd] = &ZstdEncoder{level: ce.config.DefaultLevel}
	ce.stats.AlgorithmStats[AlgorithmZstd] = &AlgorithmStats{}

	fmt.Printf("Compression: Initialized %d encoders\n", len(ce.encoders))
}

// initializeDecoders initializes compression decoders
func (ce *CompressionEngine) initializeDecoders() {
	ce.decoders[AlgorithmGzip] = &GzipDecoder{}
	ce.decoders[AlgorithmDeflate] = &DeflateDecoder{}
	ce.decoders[AlgorithmBrotli] = &BrotliDecoder{}
	ce.decoders[AlgorithmZstd] = &ZstdDecoder{}

	fmt.Printf("Compression: Initialized %d decoders\n", len(ce.decoders))
}

// Compress compresses data using the best available algorithm
func (ce *CompressionEngine) Compress(req *CompressionRequest) (*CompressionResponse, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	startTime := time.Now()
	ce.stats.TotalRequests++

	// Check if compression should be applied
	if !ce.shouldCompress(req) {
		return &CompressionResponse{
			Data:            req.Data,
			Algorithm:       AlgorithmGzip, // Default
			OriginalSize:    len(req.Data),
			CompressedSize:  len(req.Data),
			CompressionRatio: 1.0,
			Latency:         time.Since(startTime),
		}, nil
	}

	// Select best algorithm
	algorithm := ce.selectBestAlgorithm(req)
	encoder := ce.encoders[algorithm]

	// Perform compression
	compressed, err := encoder.Encode(req.Data, req.Level)
	if err != nil {
		ce.stats.ErrorCount++
		if algStats, exists := ce.stats.AlgorithmStats[algorithm]; exists {
			algStats.ErrorCount++
		}
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	// Calculate compression ratio
	originalSize := len(req.Data)
	compressedSize := len(compressed)
	ratio := float64(compressedSize) / float64(originalSize)

	// Update statistics
	ce.stats.CompressedRequests++
	ce.stats.BytesIn += uint64(originalSize)
	ce.stats.BytesOut += uint64(compressedSize)

	if algStats, exists := ce.stats.AlgorithmStats[algorithm]; exists {
		algStats.Requests++
		algStats.BytesIn += uint64(originalSize)
		algStats.BytesOut += uint64(compressedSize)
		algStats.CompressionRatio = float64(algStats.BytesOut) / float64(algStats.BytesIn)
	}

	// Update content type statistics
	ce.updateContentTypeStats(req.ContentType, originalSize, compressedSize, algorithm)

	response := &CompressionResponse{
		Data:            compressed,
		Algorithm:       algorithm,
		Level:           req.Level,
		OriginalSize:    originalSize,
		CompressedSize:  compressedSize,
		CompressionRatio: ratio,
		Latency:         time.Since(startTime),
	}

	return response, nil
}

// Decompress decompresses data
func (ce *CompressionEngine) Decompress(data []byte, algorithm Algorithm) ([]byte, error) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	decoder, exists := ce.decoders[algorithm]
	if !exists {
		return nil, fmt.Errorf("unsupported decompression algorithm: %d", algorithm)
	}

	decompressed, err := decoder.Decode(data)
	if err != nil {
		ce.stats.ErrorCount++
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	ce.stats.DecompressedRequests++
	return decompressed, nil
}

// CompressResponse compresses an HTTP response
func (ce *CompressionEngine) CompressResponse(resp *http.Response, acceptEncoding string) error {
	// Check if response can be compressed
	if !ce.canCompressResponse(resp) {
		return nil
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body.Close()

	// Create compression request
	req := &CompressionRequest{
		Data:         body,
		ContentType:  resp.Header.Get("Content-Type"),
		AcceptHeader: acceptEncoding,
		Level:        ce.config.DefaultLevel,
	}

	// Compress data
	compressed, err := ce.Compress(req)
	if err != nil {
		return fmt.Errorf("compression failed: %w", err)
	}

	// Only use compression if it provides benefit
	if compressed.CompressionRatio < ce.config.QualityThreshold {
		// Update response headers
		resp.Header.Set("Content-Encoding", ce.getAlgorithmName(compressed.Algorithm))
		resp.Header.Set("Content-Length", strconv.Itoa(compressed.CompressedSize))
		resp.Header.Del("Content-Range") // Remove range header as content changed

		// Create new body reader
		resp.Body = io.NopCloser(bytes.NewReader(compressed.Data))
		resp.ContentLength = int64(compressed.CompressedSize)
	} else {
		// Compression not beneficial, use original data
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	return nil
}

// shouldCompress determines if data should be compressed
func (ce *CompressionEngine) shouldCompress(req *CompressionRequest) bool {
	// Check size constraints
	dataSize := len(req.Data)
	if dataSize < ce.config.MinSize || dataSize > ce.config.MaxSize {
		return false
	}

	// Check content type
	if !ce.isCompressibleContentType(req.ContentType) {
		return false
	}

	return true
}

// selectBestAlgorithm selects the best compression algorithm
func (ce *CompressionEngine) selectBestAlgorithm(req *CompressionRequest) Algorithm {
	// Parse accept-encoding header
	supportedAlgorithms := ce.parseAcceptEncoding(req.AcceptHeader)

	// Find best supported algorithm
	for _, algorithm := range ce.config.EnabledAlgorithms {
		if ce.isAlgorithmSupported(algorithm, supportedAlgorithms) {
			return algorithm
		}
	}

	return ce.config.DefaultAlgorithm
}

// parseAcceptEncoding parses the Accept-Encoding header
func (ce *CompressionEngine) parseAcceptEncoding(header string) map[string]float64 {
	algorithms := make(map[string]float64)
	
	if header == "" {
		return algorithms
	}

	for _, encoding := range strings.Split(header, ",") {
		encoding = strings.TrimSpace(encoding)
		
		parts := strings.Split(encoding, ";")
		name := strings.TrimSpace(parts[0])
		quality := 1.0

		// Parse quality value
		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "q=") {
				if q, err := strconv.ParseFloat(part[2:], 64); err == nil {
					quality = q
				}
			}
		}

		algorithms[name] = quality
	}

	return algorithms
}

// isAlgorithmSupported checks if an algorithm is supported by the client
func (ce *CompressionEngine) isAlgorithmSupported(algorithm Algorithm, supported map[string]float64) bool {
	name := ce.getAlgorithmName(algorithm)
	quality, exists := supported[name]
	return exists && quality > 0
}

// getAlgorithmName returns the string name for an algorithm
func (ce *CompressionEngine) getAlgorithmName(algorithm Algorithm) string {
	switch algorithm {
	case AlgorithmGzip:
		return "gzip"
	case AlgorithmDeflate:
		return "deflate"
	case AlgorithmBrotli:
		return "br"
	case AlgorithmZstd:
		return "zstd"
	case AlgorithmLZ4:
		return "lz4"
	case AlgorithmSnappy:
		return "snappy"
	default:
		return "gzip"
	}
}

// isCompressibleContentType checks if a content type should be compressed
func (ce *CompressionEngine) isCompressibleContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	// Extract base content type
	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))

	// Check excluded types first
	for _, excluded := range ce.config.ExcludedTypes {
		if ce.matchesPattern(baseType, excluded) {
			return false
		}
	}

	// Check included types
	for _, included := range ce.config.ContentTypes {
		if ce.matchesPattern(baseType, included) {
			return true
		}
	}

	return false
}

// matchesPattern checks if a content type matches a pattern
func (ce *CompressionEngine) matchesPattern(contentType, pattern string) bool {
	if pattern == contentType {
		return true
	}

	// Handle wildcard patterns
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(contentType, prefix+"/")
	}

	return false
}

// canCompressResponse checks if an HTTP response can be compressed
func (ce *CompressionEngine) canCompressResponse(resp *http.Response) bool {
	// Check if already compressed
	if resp.Header.Get("Content-Encoding") != "" {
		return false
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if !ce.isCompressibleContentType(contentType) {
		return false
	}

	// Check content length
	if resp.ContentLength > 0 {
		if resp.ContentLength < int64(ce.config.MinSize) || resp.ContentLength > int64(ce.config.MaxSize) {
			return false
		}
	}

	return true
}

// updateContentTypeStats updates statistics for content types
func (ce *CompressionEngine) updateContentTypeStats(contentType string, originalSize, compressedSize int, algorithm Algorithm) {
	if contentType == "" {
		return
	}

	baseType := strings.Split(contentType, ";")[0]
	baseType = strings.TrimSpace(strings.ToLower(baseType))

	stats, exists := ce.stats.ContentTypeStats[baseType]
	if !exists {
		stats = &ContentTypeStats{
			OptimalAlgorithm: algorithm,
		}
		ce.stats.ContentTypeStats[baseType] = stats
	}

	stats.Requests++
	stats.BytesIn += uint64(originalSize)
	stats.BytesOut += uint64(compressedSize)
	stats.CompressionRatio = float64(stats.BytesOut) / float64(stats.BytesIn)

	// Update optimal algorithm if this one performs better
	if stats.CompressionRatio < ce.stats.AlgorithmStats[algorithm].CompressionRatio {
		stats.OptimalAlgorithm = algorithm
	}
}

// statsCollector collects compression statistics
func (ce *CompressionEngine) statsCollector() {
	ticker := time.NewTicker(ce.config.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ce.collectStatistics()
		}
	}
}

// collectStatistics collects and updates compression statistics
func (ce *CompressionEngine) collectStatistics() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Calculate overall compression ratio
	if ce.stats.BytesIn > 0 {
		ce.stats.CompressionRatio = float64(ce.stats.BytesOut) / float64(ce.stats.BytesIn)
	}

	ce.stats.LastUpdate = time.Now()
}

// Encoder implementations

// GzipEncoder implementation
func (ge *GzipEncoder) Encode(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	
	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func (ge *GzipEncoder) ContentEncoding() string {
	return "gzip"
}

func (ge *GzipEncoder) DefaultLevel() int {
	return gzip.DefaultCompression
}

func (ge *GzipEncoder) MaxLevel() int {
	return gzip.BestCompression
}

func (ge *GzipEncoder) MinLevel() int {
	return gzip.BestSpeed
}

// DeflateEncoder implementation
func (de *DeflateEncoder) Encode(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	
	writer, err := zlib.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func (de *DeflateEncoder) ContentEncoding() string {
	return "deflate"
}

func (de *DeflateEncoder) DefaultLevel() int {
	return zlib.DefaultCompression
}

func (de *DeflateEncoder) MaxLevel() int {
	return zlib.BestCompression
}

func (de *DeflateEncoder) MinLevel() int {
	return zlib.BestSpeed
}

// BrotliEncoder implementation
func (be *BrotliEncoder) Encode(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer := brotli.NewWriterLevel(&buf, level)
	
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	
	if err := writer.Close(); err != nil {
		return nil, err
	}
	
	return buf.Bytes(), nil
}

func (be *BrotliEncoder) ContentEncoding() string {
	return "br"
}

func (be *BrotliEncoder) DefaultLevel() int {
	return 6
}

func (be *BrotliEncoder) MaxLevel() int {
	return 11
}

func (be *BrotliEncoder) MinLevel() int {
	return 0
}

// ZstdEncoder implementation
func (ze *ZstdEncoder) Encode(data []byte, level int) ([]byte, error) {
	encoder, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevel(level)))
	if err != nil {
		return nil, err
	}
	defer encoder.Close()
	
	return encoder.EncodeAll(data, nil), nil
}

func (ze *ZstdEncoder) ContentEncoding() string {
	return "zstd"
}

func (ze *ZstdEncoder) DefaultLevel() int {
	return 3
}

func (ze *ZstdEncoder) MaxLevel() int {
	return 22
}

func (ze *ZstdEncoder) MinLevel() int {
	return 1
}

// Decoder implementations

// GzipDecoder implementation
func (gd *GzipDecoder) Decode(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	return io.ReadAll(reader)
}

func (gd *GzipDecoder) ContentEncoding() string {
	return "gzip"
}

// DeflateDecoder implementation
func (dd *DeflateDecoder) Decode(data []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	
	return io.ReadAll(reader)
}

func (dd *DeflateDecoder) ContentEncoding() string {
	return "deflate"
}

// BrotliDecoder implementation
func (bd *BrotliDecoder) Decode(data []byte) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(data))
	return io.ReadAll(reader)
}

func (bd *BrotliDecoder) ContentEncoding() string {
	return "br"
}

// ZstdDecoder implementation
func (zd *ZstdDecoder) Decode(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()
	
	return decoder.DecodeAll(data, nil)
}

func (zd *ZstdDecoder) ContentEncoding() string {
	return "zstd"
}

// EncoderPool implementation
func NewEncoderPool(size int) *EncoderPool {
	pool := &EncoderPool{
		pools: make(map[Algorithm]*sync.Pool),
	}

	// Initialize pools for each algorithm
	pool.pools[AlgorithmGzip] = &sync.Pool{
		New: func() interface{} {
			return &GzipEncoder{}
		},
	}

	pool.pools[AlgorithmBrotli] = &sync.Pool{
		New: func() interface{} {
			return &BrotliEncoder{}
		},
	}

	pool.pools[AlgorithmZstd] = &sync.Pool{
		New: func() interface{} {
			return &ZstdEncoder{}
		},
	}

	return pool
}

// GetEncoder gets an encoder from the pool
func (ep *EncoderPool) GetEncoder(algorithm Algorithm) Encoder {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	if pool, exists := ep.pools[algorithm]; exists {
		return pool.Get().(Encoder)
	}

	return nil
}

// PutEncoder returns an encoder to the pool
func (ep *EncoderPool) PutEncoder(algorithm Algorithm, encoder Encoder) {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	if pool, exists := ep.pools[algorithm]; exists {
		pool.Put(encoder)
	}
}

// GetStats returns compression engine statistics
func (ce *CompressionEngine) GetStats() *CompressionStats {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	stats := *ce.stats
	return &stats
}

// GetSupportedAlgorithms returns list of supported algorithms
func (ce *CompressionEngine) GetSupportedAlgorithms() []Algorithm {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	algorithms := make([]Algorithm, 0, len(ce.encoders))
	for algorithm := range ce.encoders {
		algorithms = append(algorithms, algorithm)
	}
	return algorithms
}