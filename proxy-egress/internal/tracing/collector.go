package tracing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type SpanCollector struct {
	spans      map[string]*CollectedSpan
	mutex      sync.RWMutex
	config     CollectorConfig
	exporters  []SpanExporter
	processors []SpanProcessor
	filters    []SpanFilter
	metrics    *CollectorMetrics
}

type CollectorConfig struct {
	MaxSpans           int
	RetentionPeriod   time.Duration
	BatchSize         int
	FlushInterval     time.Duration
	EnableMetrics     bool
	EnableSampling    bool
	SamplingRate      float64
}

type CollectedSpan struct {
	TraceID       string                 `json:"trace_id"`
	SpanID        string                 `json:"span_id"`
	ParentSpanID  string                 `json:"parent_span_id"`
	OperationName string                 `json:"operation_name"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      time.Duration          `json:"duration"`
	Status        string                 `json:"status"`
	Attributes    map[string]interface{} `json:"attributes"`
	Events        []SpanEvent            `json:"events"`
	Links         []SpanLink             `json:"links"`
	Resource      map[string]string      `json:"resource"`
	Tags          []string               `json:"tags"`
}

type SpanEvent struct {
	Name       string                 `json:"name"`
	Timestamp  time.Time              `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes"`
}

type SpanLink struct {
	TraceID    string                 `json:"trace_id"`
	SpanID     string                 `json:"span_id"`
	Attributes map[string]interface{} `json:"attributes"`
}

type SpanExporter interface {
	Export(ctx context.Context, spans []*CollectedSpan) error
	Shutdown(ctx context.Context) error
}

type SpanProcessor interface {
	Process(span *CollectedSpan) *CollectedSpan
	Name() string
}

type SpanFilter interface {
	ShouldInclude(span *CollectedSpan) bool
	Name() string
}

type CollectorMetrics struct {
	SpansCollected     uint64
	SpansExported      uint64
	SpansDropped       uint64
	SpansFiltered      uint64
	ExportErrors       uint64
	ProcessingErrors   uint64
	AverageLatency     time.Duration
	TotalLatency       time.Duration
	mutex              sync.RWMutex
}

type FileExporter struct {
	filename string
	config   FileExporterConfig
	mutex    sync.Mutex
}

type FileExporterConfig struct {
	Filename     string
	MaxFileSize  int64
	RotateDaily  bool
	Compression  bool
	Format       ExportFormat
}

type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatCSV  ExportFormat = "csv"
	FormatTSV  ExportFormat = "tsv"
)

type HTTPExporter struct {
	endpoint string
	config   HTTPExporterConfig
	client   HTTPClient
}

type HTTPExporterConfig struct {
	Endpoint    string
	Headers     map[string]string
	Timeout     time.Duration
	RetryCount  int
	BatchSize   int
	Compression bool
}

type HTTPClient interface {
	Post(url string, contentType string, body []byte) error
}

type DatabaseExporter struct {
	config DatabaseExporterConfig
	db     DatabaseClient
}

type DatabaseExporterConfig struct {
	ConnectionString string
	TableName       string
	BatchSize       int
	Timeout         time.Duration
	CreateTable     bool
}

type DatabaseClient interface {
	Insert(spans []*CollectedSpan) error
	Close() error
}

type AttributeProcessor struct {
	transformations map[string]AttributeTransformation
}

type AttributeTransformation func(value interface{}) interface{}

type TagProcessor struct {
	tagExtractors []TagExtractor
}

type TagExtractor func(span *CollectedSpan) []string

type DurationFilter struct {
	minDuration time.Duration
	maxDuration time.Duration
}

type StatusFilter struct {
	allowedStatuses []string
}

type AttributeFilter struct {
	requiredAttributes map[string]interface{}
}

func NewSpanCollector(config CollectorConfig) *SpanCollector {
	return &SpanCollector{
		spans:   make(map[string]*CollectedSpan),
		config:  config,
		metrics: &CollectorMetrics{},
	}
}

func (sc *SpanCollector) CollectSpan(span trace.Span, ctx context.Context) error {
	spanCtx := span.SpanContext()
	if !spanCtx.IsValid() {
		return fmt.Errorf("invalid span context")
	}

	collectedSpan := &CollectedSpan{
		TraceID:      spanCtx.TraceID().String(),
		SpanID:       spanCtx.SpanID().String(),
		StartTime:    time.Now(),
		Status:       "active",
		Attributes:   make(map[string]interface{}),
		Events:       make([]SpanEvent, 0),
		Links:        make([]SpanLink, 0),
		Resource:     make(map[string]string),
		Tags:         make([]string, 0),
	}

	if sc.config.EnableSampling && !sc.shouldSample() {
		return nil
	}

	for _, filter := range sc.filters {
		if !filter.ShouldInclude(collectedSpan) {
			sc.metrics.recordFiltered()
			return nil
		}
	}

	for _, processor := range sc.processors {
		collectedSpan = processor.Process(collectedSpan)
	}

	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if len(sc.spans) >= sc.config.MaxSpans {
		sc.evictOldestSpan()
	}

	sc.spans[collectedSpan.SpanID] = collectedSpan
	sc.metrics.recordCollected()

	return nil
}

func (sc *SpanCollector) FinishSpan(spanID string, endTime time.Time, status string) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	span, exists := sc.spans[spanID]
	if !exists {
		return fmt.Errorf("span not found: %s", spanID)
	}

	span.EndTime = endTime
	span.Duration = endTime.Sub(span.StartTime)
	span.Status = status

	sc.tryExport([]*CollectedSpan{span})
	return nil
}

func (sc *SpanCollector) AddEvent(spanID string, event SpanEvent) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	span, exists := sc.spans[spanID]
	if !exists {
		return fmt.Errorf("span not found: %s", spanID)
	}

	span.Events = append(span.Events, event)
	return nil
}

func (sc *SpanCollector) SetAttribute(spanID string, key string, value interface{}) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	span, exists := sc.spans[spanID]
	if !exists {
		return fmt.Errorf("span not found: %s", spanID)
	}

	span.Attributes[key] = value
	return nil
}

func (sc *SpanCollector) GetSpan(spanID string) (*CollectedSpan, bool) {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	span, exists := sc.spans[spanID]
	return span, exists
}

func (sc *SpanCollector) GetTraceSpans(traceID string) []*CollectedSpan {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	var spans []*CollectedSpan
	for _, span := range sc.spans {
		if span.TraceID == traceID {
			spans = append(spans, span)
		}
	}

	return spans
}

func (sc *SpanCollector) AddExporter(exporter SpanExporter) {
	sc.exporters = append(sc.exporters, exporter)
}

func (sc *SpanCollector) AddProcessor(processor SpanProcessor) {
	sc.processors = append(sc.processors, processor)
}

func (sc *SpanCollector) AddFilter(filter SpanFilter) {
	sc.filters = append(sc.filters, filter)
}

func (sc *SpanCollector) shouldSample() bool {
	return sc.config.SamplingRate >= 1.0
}

func (sc *SpanCollector) evictOldestSpan() {
	var oldest *CollectedSpan
	var oldestKey string

	for key, span := range sc.spans {
		if oldest == nil || span.StartTime.Before(oldest.StartTime) {
			oldest = span
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(sc.spans, oldestKey)
		sc.metrics.recordDropped()
	}
}

func (sc *SpanCollector) tryExport(spans []*CollectedSpan) {
	if len(sc.exporters) == 0 {
		return
	}

	for _, exporter := range sc.exporters {
		go func(exp SpanExporter) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := exp.Export(ctx, spans); err != nil {
				sc.metrics.recordExportError()
			} else {
				sc.metrics.recordExported(len(spans))
			}
		}(exporter)
	}
}

func (sc *SpanCollector) Flush(ctx context.Context) error {
	sc.mutex.RLock()
	var spans []*CollectedSpan
	for _, span := range sc.spans {
		spans = append(spans, span)
	}
	sc.mutex.RUnlock()

	if len(spans) == 0 {
		return nil
	}

	batchSize := sc.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(spans); i += batchSize {
		end := i + batchSize
		if end > len(spans) {
			end = len(spans)
		}

		batch := spans[i:end]
		sc.tryExport(batch)
	}

	return nil
}

func (sc *SpanCollector) Shutdown(ctx context.Context) error {
	if err := sc.Flush(ctx); err != nil {
		return err
	}

	for _, exporter := range sc.exporters {
		if err := exporter.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (sc *SpanCollector) GetMetrics() *CollectorMetrics {
	sc.metrics.mutex.RLock()
	defer sc.metrics.mutex.RUnlock()

	metricsCopy := *sc.metrics
	return &metricsCopy
}

func (cm *CollectorMetrics) recordCollected() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.SpansCollected++
}

func (cm *CollectorMetrics) recordExported(count int) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.SpansExported += uint64(count)
}

func (cm *CollectorMetrics) recordDropped() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.SpansDropped++
}

func (cm *CollectorMetrics) recordFiltered() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.SpansFiltered++
}

func (cm *CollectorMetrics) recordExportError() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.ExportErrors++
}

func NewAttributeProcessor(transformations map[string]AttributeTransformation) *AttributeProcessor {
	return &AttributeProcessor{
		transformations: transformations,
	}
}

func (ap *AttributeProcessor) Process(span *CollectedSpan) *CollectedSpan {
	for key, transformation := range ap.transformations {
		if value, exists := span.Attributes[key]; exists {
			span.Attributes[key] = transformation(value)
		}
	}
	return span
}

func (ap *AttributeProcessor) Name() string {
	return "attribute-processor"
}

func NewTagProcessor(extractors []TagExtractor) *TagProcessor {
	return &TagProcessor{
		tagExtractors: extractors,
	}
}

func (tp *TagProcessor) Process(span *CollectedSpan) *CollectedSpan {
	for _, extractor := range tp.tagExtractors {
		tags := extractor(span)
		span.Tags = append(span.Tags, tags...)
	}
	return span
}

func (tp *TagProcessor) Name() string {
	return "tag-processor"
}

func NewDurationFilter(minDuration, maxDuration time.Duration) *DurationFilter {
	return &DurationFilter{
		minDuration: minDuration,
		maxDuration: maxDuration,
	}
}

func (df *DurationFilter) ShouldInclude(span *CollectedSpan) bool {
	if df.minDuration > 0 && span.Duration < df.minDuration {
		return false
	}
	if df.maxDuration > 0 && span.Duration > df.maxDuration {
		return false
	}
	return true
}

func (df *DurationFilter) Name() string {
	return "duration-filter"
}

func NewStatusFilter(allowedStatuses []string) *StatusFilter {
	return &StatusFilter{
		allowedStatuses: allowedStatuses,
	}
}

func (sf *StatusFilter) ShouldInclude(span *CollectedSpan) bool {
	for _, status := range sf.allowedStatuses {
		if span.Status == status {
			return true
		}
	}
	return len(sf.allowedStatuses) == 0
}

func (sf *StatusFilter) Name() string {
	return "status-filter"
}

func HTTPMethodTagExtractor(span *CollectedSpan) []string {
	if method, exists := span.Attributes["http.method"]; exists {
		if methodStr, ok := method.(string); ok {
			return []string{"method:" + methodStr}
		}
	}
	return nil
}

func StatusCodeTagExtractor(span *CollectedSpan) []string {
	if statusCode, exists := span.Attributes["http.status_code"]; exists {
		return []string{fmt.Sprintf("status:%v", statusCode)}
	}
	return nil
}

func ErrorTagExtractor(span *CollectedSpan) []string {
	if span.Status == "error" {
		return []string{"error:true"}
	}
	return []string{"error:false"}
}

func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		MaxSpans:        10000,
		RetentionPeriod: 1 * time.Hour,
		BatchSize:       100,
		FlushInterval:   30 * time.Second,
		EnableMetrics:   true,
		EnableSampling:  true,
		SamplingRate:    0.1,
	}
}