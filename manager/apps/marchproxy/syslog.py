"""
MarchProxy Syslog Integration

UDP Syslog client for centralized logging of authentication events,
network flows, and debug information.
"""

import socket
import json
import time
from datetime import datetime
from typing import Dict, Any, Optional
import threading

class SyslogClient:
    """UDP Syslog client for MarchProxy logging"""
    
    # Syslog facilities
    FACILITY_LOCAL0 = 16
    FACILITY_LOCAL1 = 17
    FACILITY_LOCAL2 = 18
    FACILITY_LOCAL3 = 19
    FACILITY_LOCAL4 = 20
    FACILITY_LOCAL5 = 21
    FACILITY_LOCAL6 = 22
    FACILITY_LOCAL7 = 23
    
    # Syslog severities
    SEVERITY_EMERGENCY = 0      # System is unusable
    SEVERITY_ALERT = 1          # Action must be taken immediately
    SEVERITY_CRITICAL = 2       # Critical conditions
    SEVERITY_ERROR = 3          # Error conditions
    SEVERITY_WARNING = 4        # Warning conditions
    SEVERITY_NOTICE = 5         # Normal but significant condition
    SEVERITY_INFO = 6           # Informational messages
    SEVERITY_DEBUG = 7          # Debug-level messages
    
    def __init__(self, hostname: str = 'localhost', port: int = 514, 
                 facility: int = None, app_name: str = 'marchproxy'):
        """
        Initialize syslog client
        
        Args:
            hostname: Syslog server hostname
            port: Syslog server port (default 514)
            facility: Syslog facility (default LOCAL0)
            app_name: Application name for syslog messages
        """
        self.hostname = hostname
        self.port = port
        self.facility = facility or self.FACILITY_LOCAL0
        self.app_name = app_name
        self.socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.lock = threading.Lock()
    
    def _calculate_priority(self, severity: int) -> int:
        """Calculate syslog priority from facility and severity"""
        return self.facility * 8 + severity
    
    def _format_message(self, severity: int, message: str, 
                       structured_data: Dict[str, Any] = None) -> str:
        """Format message according to RFC 5424"""
        priority = self._calculate_priority(severity)
        timestamp = datetime.utcnow().isoformat() + 'Z'
        hostname = socket.gethostname()
        
        # Base message format: <priority>version timestamp hostname app-name procid msgid msg
        base_msg = f"<{priority}>1 {timestamp} {hostname} {self.app_name} - - "
        
        # Add structured data if provided
        if structured_data:
            sd_str = ""
            for sd_id, sd_params in structured_data.items():
                sd_str += f"[{sd_id}"
                for key, value in sd_params.items():
                    # Escape special characters
                    escaped_value = str(value).replace('"', '\\"').replace('\\', '\\\\').replace(']', '\\]')
                    sd_str += f' {key}="{escaped_value}"'
                sd_str += "]"
            base_msg += sd_str + " "
        else:
            base_msg += "- "
        
        base_msg += message
        return base_msg
    
    def send_log(self, severity: int, message: str, 
                structured_data: Dict[str, Any] = None) -> bool:
        """
        Send a log message to syslog server
        
        Args:
            severity: Syslog severity level
            message: Log message
            structured_data: Additional structured data
            
        Returns:
            True if sent successfully, False otherwise
        """
        try:
            formatted_message = self._format_message(severity, message, structured_data)
            
            with self.lock:
                self.socket.sendto(
                    formatted_message.encode('utf-8'),
                    (self.hostname, self.port)
                )
            return True
            
        except Exception as e:
            # Log to local system if syslog fails
            print(f"Syslog send failed: {e}")
            return False
    
    def log_auth_event(self, event_type: str, username: str = None, 
                      user_id: int = None, ip_address: str = None, 
                      success: bool = True, details: Dict[str, Any] = None):
        """Log authentication events"""
        severity = self.SEVERITY_INFO if success else self.SEVERITY_WARNING
        
        message = f"AUTH {event_type.upper()}: "
        if username:
            message += f"user={username} "
        if user_id:
            message += f"user_id={user_id} "
        if ip_address:
            message += f"ip={ip_address} "
        message += f"success={success}"
        
        structured_data = {
            "marchproxy": {
                "event_type": "authentication",
                "auth_event": event_type,
                "success": success,
                "timestamp": int(time.time())
            }
        }
        
        if username:
            structured_data["marchproxy"]["username"] = username
        if user_id:
            structured_data["marchproxy"]["user_id"] = user_id
        if ip_address:
            structured_data["marchproxy"]["ip_address"] = ip_address
        if details:
            structured_data["marchproxy"].update(details)
        
        self.send_log(severity, message, structured_data)
    
    def log_netflow_event(self, source_ip: str, dest_ip: str, 
                         source_port: int, dest_port: int, 
                         protocol: str, bytes_transferred: int = 0,
                         duration: float = 0, service_name: str = None):
        """Log network flow events"""
        message = f"NETFLOW: {source_ip}:{source_port} -> {dest_ip}:{dest_port} " \
                 f"proto={protocol} bytes={bytes_transferred} duration={duration:.2f}s"
        
        if service_name:
            message += f" service={service_name}"
        
        structured_data = {
            "marchproxy": {
                "event_type": "netflow",
                "source_ip": source_ip,
                "dest_ip": dest_ip,
                "source_port": source_port,
                "dest_port": dest_port,
                "protocol": protocol,
                "bytes_transferred": bytes_transferred,
                "duration": duration,
                "timestamp": int(time.time())
            }
        }
        
        if service_name:
            structured_data["marchproxy"]["service_name"] = service_name
        
        self.send_log(self.SEVERITY_INFO, message, structured_data)
    
    def log_debug_event(self, component: str, message: str, 
                       details: Dict[str, Any] = None):
        """Log debug events"""
        log_message = f"DEBUG {component.upper()}: {message}"
        
        structured_data = {
            "marchproxy": {
                "event_type": "debug",
                "component": component,
                "timestamp": int(time.time())
            }
        }
        
        if details:
            structured_data["marchproxy"].update(details)
        
        self.send_log(self.SEVERITY_DEBUG, log_message, structured_data)
    
    def log_system_event(self, event_type: str, message: str, 
                        severity: int = None, details: Dict[str, Any] = None):
        """Log general system events"""
        if severity is None:
            severity = self.SEVERITY_INFO
        
        log_message = f"SYSTEM {event_type.upper()}: {message}"
        
        structured_data = {
            "marchproxy": {
                "event_type": "system",
                "system_event": event_type,
                "timestamp": int(time.time())
            }
        }
        
        if details:
            structured_data["marchproxy"].update(details)
        
        self.send_log(severity, log_message, structured_data)
    
    def close(self):
        """Close the syslog connection"""
        if self.socket:
            self.socket.close()


class ClusterSyslogManager:
    """Manages syslog clients for different clusters"""
    
    def __init__(self):
        self.clients = {}  # cluster_id -> SyslogClient
        self.lock = threading.Lock()
    
    def get_client(self, cluster_id: int, syslog_endpoint: str = None) -> Optional[SyslogClient]:
        """Get or create syslog client for cluster"""
        if not syslog_endpoint:
            return None
        
        try:
            # Parse hostname:port from endpoint
            if ':' in syslog_endpoint:
                hostname, port_str = syslog_endpoint.rsplit(':', 1)
                port = int(port_str)
            else:
                hostname = syslog_endpoint
                port = 514
            
            with self.lock:
                client_key = f"{cluster_id}_{hostname}_{port}"
                
                if client_key not in self.clients:
                    self.clients[client_key] = SyslogClient(
                        hostname=hostname,
                        port=port,
                        app_name=f"marchproxy-cluster-{cluster_id}"
                    )
                
                return self.clients[client_key]
                
        except Exception as e:
            print(f"Failed to create syslog client: {e}")
            return None
    
    def log_to_cluster(self, cluster_id: int, syslog_endpoint: str,
                      log_type: str, **kwargs):
        """Send log to cluster's syslog endpoint"""
        client = self.get_client(cluster_id, syslog_endpoint)
        if not client:
            return
        
        if log_type == 'auth':
            client.log_auth_event(**kwargs)
        elif log_type == 'netflow':
            client.log_netflow_event(**kwargs)
        elif log_type == 'debug':
            client.log_debug_event(**kwargs)
        elif log_type == 'system':
            client.log_system_event(**kwargs)
    
    def close_all(self):
        """Close all syslog connections"""
        with self.lock:
            for client in self.clients.values():
                client.close()
            self.clients.clear()


# Global cluster syslog manager
cluster_syslog_manager = ClusterSyslogManager()