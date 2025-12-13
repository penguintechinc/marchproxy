// MarchProxy Metrics Filter (WASM)
// Custom metrics collection for MarchProxy

use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use serde::{Deserialize, Serialize};

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Info);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(MetricsFilterRoot {
            config: FilterConfig::default(),
        })
    });
}}

#[derive(Debug, Clone, Deserialize, Serialize)]
struct FilterConfig {
    enable_request_metrics: bool,
    enable_response_metrics: bool,
    enable_timing_metrics: bool,
    enable_size_metrics: bool,
    sample_rate: f32,
}

impl Default for FilterConfig {
    fn default() -> Self {
        Self {
            enable_request_metrics: true,
            enable_response_metrics: true,
            enable_timing_metrics: true,
            enable_size_metrics: true,
            sample_rate: 1.0,
        }
    }
}

struct MetricsFilterRoot {
    config: FilterConfig,
}

impl Context for MetricsFilterRoot {}

impl RootContext for MetricsFilterRoot {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        if let Some(config_bytes) = self.get_plugin_configuration() {
            match serde_json::from_slice::<FilterConfig>(&config_bytes) {
                Ok(config) => {
                    self.config = config;
                    proxy_wasm::hostcalls::log(LogLevel::Info, &format!("Metrics filter configured - sample rate: {}", self.config.sample_rate)).ok();
                    true
                }
                Err(e) => {
                    proxy_wasm::hostcalls::log(LogLevel::Error, &format!("Failed to parse metrics configuration: {}", e)).ok();
                    false
                }
            }
        } else {
            proxy_wasm::hostcalls::log(LogLevel::Info, &format!("No metrics configuration provided, using defaults")).ok();
            true
        }
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(MetricsFilter {
            config: self.config.clone(),
            request_start_time: 0,
            request_size: 0,
            response_size: 0,
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

struct MetricsFilter {
    config: FilterConfig,
    request_start_time: u64,
    request_size: usize,
    response_size: usize,
}

impl Context for MetricsFilter {}

impl HttpContext for MetricsFilter {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // Record request start time
        self.request_start_time = self.get_current_time().duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default().as_nanos() as u64;

        // Skip metrics collection based on sample rate
        if !self.should_sample() {
            return Action::Continue;
        }

        if self.config.enable_request_metrics {
            // Get request details
            let method = self.get_http_request_header(":method").unwrap_or_default();
            let path = self.get_http_request_header(":path").unwrap_or_default();
            let host = self.get_http_request_header(":authority").unwrap_or_default();
            let user_agent = self.get_http_request_header("user-agent").unwrap_or_default();

            // Increment request counter
            self.increment_metric("marchproxy_requests_total", 1);

            // Record request by method
            let metric_name = format!("marchproxy_requests_by_method_{}", method.to_lowercase());
            self.increment_metric(&metric_name, 1);

            // Record request by path (sanitized)
            let path_prefix = self.get_path_prefix(&path);
            let metric_name = format!("marchproxy_requests_by_path_{}", path_prefix);
            self.increment_metric(&metric_name, 1);

            proxy_wasm::hostcalls::log(LogLevel::Debug, &format!("Request: {} {} from {}", method, path, host)).ok();
        }

        Action::Continue
    }

    fn on_http_request_body(&mut self, body_size: usize, _end_of_stream: bool) -> Action {
        if self.config.enable_size_metrics && self.should_sample() {
            self.request_size += body_size;
        }
        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        if !self.should_sample() {
            return Action::Continue;
        }

        if self.config.enable_response_metrics {
            // Get response status
            let status = self.get_http_response_header(":status").unwrap_or_default();
            let status_code: u32 = status.parse().unwrap_or(0);

            // Increment response counter
            self.increment_metric("marchproxy_responses_total", 1);

            // Record by status code
            let metric_name = format!("marchproxy_responses_by_status_{}", status_code);
            self.increment_metric(&metric_name, 1);

            // Record by status class (2xx, 3xx, 4xx, 5xx)
            let status_class = status_code / 100;
            let metric_name = format!("marchproxy_responses_by_class_{}xx", status_class);
            self.increment_metric(&metric_name, 1);

            proxy_wasm::hostcalls::log(LogLevel::Debug, &format!("Response: {}", status_code)).ok();
        }

        if self.config.enable_timing_metrics {
            // Calculate request duration
            let now = self.get_current_time().duration_since(std::time::UNIX_EPOCH)
                .unwrap_or_default().as_nanos() as u64;
            let duration_ns = now - self.request_start_time;
            let duration_ms = duration_ns as f64 / 1_000_000.0;

            // Record latency histogram
            self.record_metric("marchproxy_request_duration_ms", duration_ms as u64);

            proxy_wasm::hostcalls::log(LogLevel::Debug, &format!("Request duration: {:.2}ms", duration_ms)).ok();
        }

        Action::Continue
    }

    fn on_http_response_body(&mut self, body_size: usize, _end_of_stream: bool) -> Action {
        if self.config.enable_size_metrics && self.should_sample() {
            self.response_size += body_size;
        }
        Action::Continue
    }

    fn on_log(&mut self) {
        if !self.should_sample() {
            return;
        }

        if self.config.enable_size_metrics {
            // Record request and response sizes
            if self.request_size > 0 {
                self.record_metric("marchproxy_request_size_bytes", self.request_size as u64);
            }
            if self.response_size > 0 {
                self.record_metric("marchproxy_response_size_bytes", self.response_size as u64);
            }

            proxy_wasm::hostcalls::log(
                LogLevel::Debug,
                &format!("Request size: {} bytes, Response size: {} bytes",
                        self.request_size, self.response_size)
            ).ok();
        }
    }
}

impl MetricsFilter {
    fn should_sample(&self) -> bool {
        if self.config.sample_rate >= 1.0 {
            return true;
        }

        // Simple sampling: use current time for pseudo-random sampling
        let now = self.get_current_time().duration_since(std::time::UNIX_EPOCH)
            .unwrap_or_default().as_millis() as u64;
        let sample_threshold = (self.config.sample_rate * 1000.0) as u64;
        (now % 1000) < sample_threshold
    }

    fn get_path_prefix(&self, path: &str) -> String {
        // Extract first path component for grouping
        let parts: Vec<&str> = path.split('/').filter(|s| !s.is_empty()).collect();
        if parts.is_empty() {
            return "root".to_string();
        }

        // Return first path component, sanitized
        parts[0].chars()
            .filter(|c| c.is_alphanumeric() || *c == '-' || *c == '_')
            .collect()
    }

    fn increment_metric(&self, name: &str, value: u64) {
        // Use Envoy's metric system
        // Note: In a real implementation, this would use the Envoy stats system
        // For WASM, we rely on Envoy's built-in metrics collection
        proxy_wasm::hostcalls::log(LogLevel::Trace, &format!("Metric: {} += {}", name, value)).ok();
    }

    fn record_metric(&self, name: &str, value: u64) {
        // Record histogram/gauge metric
        proxy_wasm::hostcalls::log(LogLevel::Trace, &format!("Metric: {} = {}", name, value)).ok();
    }
}
