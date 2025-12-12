// MarchProxy License Filter (WASM)
// Enterprise feature gating based on license validation

use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Info);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(LicenseFilterRoot {
            config: FilterConfig::default(),
        })
    });
}}

#[derive(Debug, Clone, Deserialize, Serialize)]
struct FilterConfig {
    license_key: String,
    is_enterprise: bool,
    features: HashMap<String, bool>,
    max_proxies: u32,
    current_proxies: u32,
}

impl Default for FilterConfig {
    fn default() -> Self {
        let mut features = HashMap::new();
        features.insert("basic_proxy".to_string(), true);
        features.insert("rate_limiting".to_string(), false);
        features.insert("advanced_routing".to_string(), false);
        features.insert("multi_cloud".to_string(), false);
        features.insert("distributed_tracing".to_string(), false);
        features.insert("zero_trust".to_string(), false);

        Self {
            license_key: String::from("COMMUNITY"),
            is_enterprise: false,
            features,
            max_proxies: 3,
            current_proxies: 0,
        }
    }
}

struct LicenseFilterRoot {
    config: FilterConfig,
}

impl Context for LicenseFilterRoot {}

impl RootContext for LicenseFilterRoot {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        if let Some(config_bytes) = self.get_plugin_configuration() {
            match serde_json::from_slice::<FilterConfig>(&config_bytes) {
                Ok(config) => {
                    self.config = config;
                    log::info!("License filter configured - Edition: {}",
                              if self.config.is_enterprise { "Enterprise" } else { "Community" });
                    log::info!("License: {}", self.config.license_key);
                    log::info!("Max proxies: {}", self.config.max_proxies);
                    true
                }
                Err(e) => {
                    log::error!("Failed to parse license configuration: {}", e);
                    false
                }
            }
        } else {
            log::info!("No license configuration provided, using Community defaults");
            true
        }
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(LicenseFilter {
            config: self.config.clone(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

struct LicenseFilter {
    config: FilterConfig,
}

impl Context for LicenseFilter {}

impl HttpContext for LicenseFilter {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // Get request path to determine which feature is being accessed
        let path = self.get_http_request_header(":path").unwrap_or_default();

        // Check for enterprise feature paths
        let required_feature = self.get_required_feature(&path);

        if let Some(feature) = required_feature {
            if !self.is_feature_enabled(&feature) {
                log::warn!("Feature '{}' not available in current license", feature);
                self.send_http_response(
                    402,
                    vec![
                        ("content-type", "application/json"),
                        ("x-license-required", "enterprise"),
                    ],
                    Some(format!(
                        "{{\"error\":\"Enterprise license required for feature: {}\",\"upgrade_url\":\"https://marchproxy.penguintech.io/pricing\"}}",
                        feature
                    ).as_bytes()),
                );
                return Action::Pause;
            }
        }

        // Check proxy count limit
        if self.config.current_proxies > self.config.max_proxies {
            log::error!("Proxy count ({}) exceeds license limit ({})",
                       self.config.current_proxies, self.config.max_proxies);
            self.send_http_response(
                429,
                vec![
                    ("content-type", "application/json"),
                    ("x-license-limit-exceeded", "true"),
                ],
                Some(format!(
                    "{{\"error\":\"Proxy count limit exceeded\",\"current\":{},\"limit\":{},\"upgrade_url\":\"https://marchproxy.penguintech.io/pricing\"}}",
                    self.config.current_proxies, self.config.max_proxies
                ).as_bytes()),
            );
            return Action::Pause;
        }

        // Add license information to request headers
        self.set_http_request_header("x-license-edition",
                                    if self.config.is_enterprise { "enterprise" } else { "community" });
        self.set_http_request_header("x-license-key", &self.config.license_key);

        Action::Continue
    }

    fn on_http_response_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // Add license information to response headers
        self.set_http_response_header("x-marchproxy-edition",
                                     if self.config.is_enterprise { "enterprise" } else { "community" });

        Action::Continue
    }
}

impl LicenseFilter {
    fn get_required_feature(&self, path: &str) -> Option<String> {
        // Map paths to required enterprise features
        if path.starts_with("/api/v1/traffic-shaping") {
            Some("advanced_routing".to_string())
        } else if path.starts_with("/api/v1/multi-cloud") {
            Some("multi_cloud".to_string())
        } else if path.starts_with("/api/v1/tracing") {
            Some("distributed_tracing".to_string())
        } else if path.starts_with("/api/v1/zero-trust") {
            Some("zero_trust".to_string())
        } else if path.starts_with("/api/v1/advanced-rate-limit") {
            Some("rate_limiting".to_string())
        } else {
            None
        }
    }

    fn is_feature_enabled(&self, feature: &str) -> bool {
        self.config.features.get(feature).copied().unwrap_or(false)
    }
}
