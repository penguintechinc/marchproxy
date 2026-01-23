"""
KillKrill service for sending logs and metrics from MarchProxy Manager

Copyright (C) 2025 MarchProxy Contributors
Licensed under GNU Affero General Public License v3.0
"""

import json
import time
import threading
import logging
import os
from typing import Dict, List, Any, Optional
from datetime import datetime, timezone
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry


class KillKrillService:
    """Service for sending logs and metrics to KillKrill"""

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.enabled = config.get("enabled", False)
        self.log_endpoint = config.get("log_endpoint", "")
        self.metrics_endpoint = config.get("metrics_endpoint", "")
        self.api_key = config.get("api_key", "")
        self.source_name = config.get("source_name", "marchproxy-manager")
        self.application = config.get("application", "manager")
        self.batch_size = config.get("batch_size", 50)
        self.flush_interval = config.get("flush_interval", 10)
        self.timeout = config.get("timeout", 30)
        self.use_http3 = config.get("use_http3", False)

        self.log_buffer = []
        self.metric_buffer = []
        self.log_lock = threading.Lock()
        self.metric_lock = threading.Lock()
        self.stop_event = threading.Event()
        self.flush_thread = None

        # Setup HTTP session with retries
        self.session = requests.Session()
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504],
            allowed_methods=["POST"],
        )
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)

        # Set default headers
        self.session.headers.update(
            {
                "Content-Type": "application/json",
                "X-API-Key": self.api_key,
                "User-Agent": "marchproxy-manager/1.0.0",
            }
        )

        if self.enabled:
            self.start_flush_thread()

    def start_flush_thread(self):
        """Start the background flush thread"""
        if self.flush_thread is None or not self.flush_thread.is_alive():
            self.flush_thread = threading.Thread(target=self._flush_loop, daemon=True)
            self.flush_thread.start()

    def stop(self):
        """Stop the service and flush remaining data"""
        if not self.enabled:
            return

        self.stop_event.set()
        if self.flush_thread and self.flush_thread.is_alive():
            self.flush_thread.join(timeout=5)

        # Flush any remaining data
        self.flush_logs()
        self.flush_metrics()

    def send_log(self, level: str, message: str, **kwargs):
        """Send a log entry to KillKrill"""
        if not self.enabled:
            return

        hostname = kwargs.get("hostname", os.uname().nodename)
        timestamp = kwargs.get("timestamp", datetime.now(timezone.utc).isoformat())

        # Extract labels and tags
        labels = {}
        tags = kwargs.get("tags", [])

        for key, value in kwargs.items():
            if key not in [
                "hostname",
                "timestamp",
                "tags",
                "trace_id",
                "span_id",
                "transaction_id",
            ]:
                labels[key] = value

        log_entry = {
            "timestamp": timestamp,
            "log_level": level.lower(),
            "message": message,
            "service_name": "marchproxy-manager",
            "hostname": hostname,
            "logger_name": kwargs.get("logger_name", "manager"),
            "thread_name": kwargs.get("thread_name", threading.current_thread().name),
            "ecs_version": "8.0",
            "labels": labels,
            "tags": tags,
        }

        # Add trace information if available
        if "trace_id" in kwargs:
            log_entry["trace_id"] = kwargs["trace_id"]
        if "span_id" in kwargs:
            log_entry["span_id"] = kwargs["span_id"]
        if "transaction_id" in kwargs:
            log_entry["transaction_id"] = kwargs["transaction_id"]

        with self.log_lock:
            self.log_buffer.append(log_entry)
            if len(self.log_buffer) >= self.batch_size:
                self._flush_logs()

    def send_metric(
        self,
        name: str,
        metric_type: str,
        value: float,
        labels: Optional[Dict[str, str]] = None,
        help_text: Optional[str] = None,
    ):
        """Send a metric to KillKrill"""
        if not self.enabled:
            return

        metric_entry = {
            "name": name,
            "type": metric_type,
            "value": value,
            "labels": labels or {},
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "help": help_text or "",
        }

        with self.metric_lock:
            self.metric_buffer.append(metric_entry)
            if len(self.metric_buffer) >= self.batch_size:
                self._flush_metrics()

    def flush_logs(self):
        """Flush log buffer"""
        with self.log_lock:
            self._flush_logs()

    def flush_metrics(self):
        """Flush metrics buffer"""
        with self.metric_lock:
            self._flush_metrics()

    def _flush_loop(self):
        """Background thread for periodic flushing"""
        while not self.stop_event.is_set():
            try:
                self.stop_event.wait(self.flush_interval)
                if not self.stop_event.is_set():
                    self.flush_logs()
                    self.flush_metrics()
            except Exception as e:
                logging.error(f"Error in KillKrill flush loop: {e}")

    def _flush_logs(self):
        """Internal log flush (must hold log_lock)"""
        if not self.log_buffer:
            return

        batch = {
            "source": self.source_name,
            "application": self.application,
            "logs": self.log_buffer.copy(),
        }
        self.log_buffer.clear()

        # Send in background thread to avoid blocking
        threading.Thread(
            target=self._send_log_batch, args=(batch,), daemon=True
        ).start()

    def _flush_metrics(self):
        """Internal metrics flush (must hold metric_lock)"""
        if not self.metric_buffer:
            return

        batch = {"source": self.source_name, "metrics": self.metric_buffer.copy()}
        self.metric_buffer.clear()

        # Send in background thread to avoid blocking
        threading.Thread(
            target=self._send_metric_batch, args=(batch,), daemon=True
        ).start()

    def _send_log_batch(self, batch: Dict[str, Any]):
        """Send log batch to KillKrill"""
        try:
            response = self.session.post(
                self.log_endpoint, json=batch, timeout=self.timeout
            )
            response.raise_for_status()
        except Exception as e:
            logging.warning(f"Failed to send log batch to KillKrill: {e}")
            # TODO: Consider implementing retry logic or dead letter queue

    def _send_metric_batch(self, batch: Dict[str, Any]):
        """Send metric batch to KillKrill"""
        try:
            response = self.session.post(
                self.metrics_endpoint, json=batch, timeout=self.timeout
            )
            response.raise_for_status()
        except Exception as e:
            logging.warning(f"Failed to send metric batch to KillKrill: {e}")
            # TODO: Consider implementing retry logic or dead letter queue

    def health_check(self) -> Dict[str, Any]:
        """Perform health check on KillKrill endpoints"""
        if not self.enabled:
            return {"status": "disabled"}

        results = {"status": "healthy", "endpoints": {}}

        # Check log endpoint
        try:
            response = self.session.get(
                f"{self.log_endpoint.replace('/api/v1/logs', '/healthz')}", timeout=5
            )
            results["endpoints"]["logs"] = {
                "status": "healthy" if response.status_code == 200 else "unhealthy",
                "status_code": response.status_code,
                "response_time_ms": int(response.elapsed.total_seconds() * 1000),
            }
        except Exception as e:
            results["endpoints"]["logs"] = {"status": "unhealthy", "error": str(e)}

        # Check metrics endpoint
        try:
            response = self.session.get(
                f"{self.metrics_endpoint.replace('/api/v1/metrics', '/healthz')}",
                timeout=5,
            )
            results["endpoints"]["metrics"] = {
                "status": "healthy" if response.status_code == 200 else "unhealthy",
                "status_code": response.status_code,
                "response_time_ms": int(response.elapsed.total_seconds() * 1000),
            }
        except Exception as e:
            results["endpoints"]["metrics"] = {"status": "unhealthy", "error": str(e)}

        # Overall status
        if any(ep.get("status") == "unhealthy" for ep in results["endpoints"].values()):
            results["status"] = "unhealthy"

        return results

    def get_stats(self) -> Dict[str, Any]:
        """Get service statistics"""
        return {
            "enabled": self.enabled,
            "log_buffer_size": len(self.log_buffer),
            "metric_buffer_size": len(self.metric_buffer),
            "config": {
                "batch_size": self.batch_size,
                "flush_interval": self.flush_interval,
                "source_name": self.source_name,
                "application": self.application,
            },
        }


class KillKrillLogHandler(logging.Handler):
    """Custom logging handler that sends logs to KillKrill"""

    def __init__(self, killkrill_service: KillKrillService):
        super().__init__()
        self.killkrill_service = killkrill_service

    def emit(self, record):
        """Emit a log record to KillKrill"""
        try:
            # Format the message
            msg = self.format(record)

            # Extract additional fields
            labels = {}
            if hasattr(record, "__dict__"):
                for key, value in record.__dict__.items():
                    if key not in [
                        "name",
                        "msg",
                        "args",
                        "levelname",
                        "levelno",
                        "pathname",
                        "filename",
                        "module",
                        "exc_info",
                        "exc_text",
                        "stack_info",
                        "lineno",
                        "funcName",
                        "created",
                        "msecs",
                        "relativeCreated",
                        "thread",
                        "threadName",
                        "processName",
                        "process",
                        "getMessage",
                        "message",
                    ]:
                        labels[key] = value

            # Send to KillKrill
            self.killkrill_service.send_log(
                level=record.levelname,
                message=msg,
                logger_name=record.name,
                thread_name=record.threadName,
                **labels,
            )
        except Exception:
            self.handleError(record)


# Global KillKrill service instance
_killkrill_service = None


def init_killkrill_service(config: Dict[str, Any]) -> KillKrillService:
    """Initialize the global KillKrill service"""
    global _killkrill_service
    _killkrill_service = KillKrillService(config)
    return _killkrill_service


def get_killkrill_service() -> Optional[KillKrillService]:
    """Get the global KillKrill service instance"""
    return _killkrill_service


def setup_killkrill_logging(
    logger: logging.Logger, killkrill_service: KillKrillService
):
    """Setup KillKrill logging for a logger"""
    if killkrill_service and killkrill_service.enabled:
        handler = KillKrillLogHandler(killkrill_service)
        handler.setLevel(logging.DEBUG)
        logger.addHandler(handler)
