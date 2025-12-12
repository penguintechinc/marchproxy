// MarchProxy Authentication Filter (WASM)
// Validates JWT and Base64 tokens for service-to-service authentication

use proxy_wasm::traits::*;
use proxy_wasm::types::*;
use serde::{Deserialize, Serialize};
use std::time::Duration;

proxy_wasm::main! {{
    proxy_wasm::set_log_level(LogLevel::Info);
    proxy_wasm::set_root_context(|_| -> Box<dyn RootContext> {
        Box::new(AuthFilterRoot {
            config: FilterConfig::default(),
        })
    });
}}

#[derive(Debug, Clone, Deserialize, Serialize)]
struct FilterConfig {
    jwt_secret: String,
    jwt_algorithm: String,
    require_auth: bool,
    base64_tokens: Vec<String>,
    exempt_paths: Vec<String>,
}

impl Default for FilterConfig {
    fn default() -> Self {
        Self {
            jwt_secret: String::from(""),
            jwt_algorithm: String::from("HS256"),
            require_auth: true,
            base64_tokens: Vec::new(),
            exempt_paths: vec![
                String::from("/healthz"),
                String::from("/metrics"),
                String::from("/ready"),
            ],
        }
    }
}

struct AuthFilterRoot {
    config: FilterConfig,
}

impl Context for AuthFilterRoot {}

impl RootContext for AuthFilterRoot {
    fn on_configure(&mut self, _plugin_configuration_size: usize) -> bool {
        if let Some(config_bytes) = self.get_plugin_configuration() {
            match serde_json::from_slice::<FilterConfig>(&config_bytes) {
                Ok(config) => {
                    self.config = config;
                    log::info!("Auth filter configured successfully");
                    true
                }
                Err(e) => {
                    log::error!("Failed to parse configuration: {}", e);
                    false
                }
            }
        } else {
            log::info!("No configuration provided, using defaults");
            true
        }
    }

    fn create_http_context(&self, _context_id: u32) -> Option<Box<dyn HttpContext>> {
        Some(Box::new(AuthFilter {
            config: self.config.clone(),
        }))
    }

    fn get_type(&self) -> Option<ContextType> {
        Some(ContextType::HttpContext)
    }
}

struct AuthFilter {
    config: FilterConfig,
}

impl Context for AuthFilter {}

impl HttpContext for AuthFilter {
    fn on_http_request_headers(&mut self, _num_headers: usize, _end_of_stream: bool) -> Action {
        // Get request path
        let path = self.get_http_request_header(":path").unwrap_or_default();

        // Check if path is exempt from authentication
        for exempt_path in &self.config.exempt_paths {
            if path.starts_with(exempt_path) {
                log::debug!("Path {} is exempt from authentication", path);
                return Action::Continue;
            }
        }

        // If authentication is not required, pass through
        if !self.config.require_auth {
            return Action::Continue;
        }

        // Get Authorization header
        let auth_header = match self.get_http_request_header("authorization") {
            Some(header) => header,
            None => {
                log::warn!("Missing Authorization header for path: {}", path);
                self.send_http_response(
                    401,
                    vec![("content-type", "application/json")],
                    Some(b"{\"error\":\"Missing Authorization header\"}"),
                );
                return Action::Pause;
            }
        };

        // Parse authorization header
        if auth_header.starts_with("Bearer ") {
            let token = &auth_header[7..];

            // Try JWT validation first
            if self.validate_jwt(token) {
                log::debug!("JWT token validated successfully");
                return Action::Continue;
            }

            // Try Base64 token validation
            if self.validate_base64(token) {
                log::debug!("Base64 token validated successfully");
                return Action::Continue;
            }

            log::warn!("Invalid token for path: {}", path);
            self.send_http_response(
                403,
                vec![("content-type", "application/json")],
                Some(b"{\"error\":\"Invalid authentication token\"}"),
            );
            Action::Pause
        } else {
            log::warn!("Invalid Authorization header format for path: {}", path);
            self.send_http_response(
                401,
                vec![("content-type", "application/json")],
                Some(b"{\"error\":\"Invalid Authorization header format. Use: Bearer <token>\"}"),
            );
            Action::Pause
        }
    }
}

impl AuthFilter {
    fn validate_jwt(&self, token: &str) -> bool {
        if self.config.jwt_secret.is_empty() {
            return false;
        }

        use jsonwebtoken::{decode, Algorithm, DecodingKey, Validation};

        let algorithm = match self.config.jwt_algorithm.as_str() {
            "HS256" => Algorithm::HS256,
            "HS384" => Algorithm::HS384,
            "HS512" => Algorithm::HS512,
            _ => Algorithm::HS256,
        };

        let mut validation = Validation::new(algorithm);
        validation.validate_exp = true;
        validation.leeway = 60; // 60 seconds leeway for clock skew

        match decode::<serde_json::Value>(
            token,
            &DecodingKey::from_secret(self.config.jwt_secret.as_bytes()),
            &validation,
        ) {
            Ok(_) => {
                log::debug!("JWT token validation successful");
                true
            }
            Err(e) => {
                log::debug!("JWT token validation failed: {}", e);
                false
            }
        }
    }

    fn validate_base64(&self, token: &str) -> bool {
        // Check if token matches any configured base64 tokens
        for valid_token in &self.config.base64_tokens {
            if token == valid_token {
                return true;
            }
        }

        // Try to decode as base64 and compare
        if let Ok(decoded) = base64::decode(token) {
            for valid_token in &self.config.base64_tokens {
                if let Ok(valid_decoded) = base64::decode(valid_token) {
                    if decoded == valid_decoded {
                        return true;
                    }
                }
            }
        }

        false
    }
}
