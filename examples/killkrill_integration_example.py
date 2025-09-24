#!/usr/bin/env python3
"""
Example demonstrating KillKrill integration with MarchProxy Manager
"""

import logging
import time
import sys
import os

# Add the manager directory to the path for imports
sys.path.append(os.path.join(os.path.dirname(__file__), '..', 'manager'))

from services.killkrill_service import KillKrillService, KillKrillLogHandler, setup_killkrill_logging


def main():
    print("MarchProxy Manager KillKrill Integration Example")

    # Example configuration
    config = {
        'enabled': True,
        'log_endpoint': 'https://killkrill.example.com/api/v1/logs',
        'metrics_endpoint': 'https://killkrill.example.com/api/v1/metrics',
        'api_key': 'your-api-key-here',
        'source_name': 'marchproxy-manager-example',
        'application': 'manager',
        'batch_size': 5,  # Small batch for demo
        'flush_interval': 3,  # Quick flush for demo
        'timeout': 10,
        'use_http3': True
    }

    # Create KillKrill service
    killkrill_service = KillKrillService(config)

    # Setup logging
    logger = logging.getLogger('marchproxy.example')
    logger.setLevel(logging.DEBUG)

    # Add console handler
    console_handler = logging.StreamHandler()
    console_handler.setLevel(logging.INFO)
    formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
    console_handler.setFormatter(formatter)
    logger.addHandler(console_handler)

    # Setup KillKrill logging
    setup_killkrill_logging(logger, killkrill_service)

    print("Starting logging examples...")

    # Example logging
    logger.info("Manager starting with KillKrill integration",
                extra={'component': 'example', 'version': '1.0.0'})

    logger.warning("This is a warning message",
                   extra={'user_id': '12345', 'action': 'login_attempt'})

    logger.error("Example error message",
                 extra={'error_code': 'AUTH_FAILED', 'component': 'authentication'})

    # Direct KillKrill usage
    print("Sending direct logs and metrics to KillKrill...")

    killkrill_service.send_log(
        level='info',
        message='Direct KillKrill log from manager',
        component='example',
        action='demo',
        tags=['example', 'demo']
    )

    killkrill_service.send_metric(
        name='manager_example_counter',
        metric_type='counter',
        value=1.0,
        labels={'component': 'example', 'type': 'demo'},
        help_text='Example counter from manager'
    )

    killkrill_service.send_metric(
        name='manager_active_sessions',
        metric_type='gauge',
        value=42.0,
        labels={'instance': 'example'},
        help_text='Number of active sessions'
    )

    # Health check example
    print("\nPerforming KillKrill health check...")
    health = killkrill_service.health_check()
    print(f"Health check result: {health}")

    # Stats example
    print("\nGetting KillKrill service stats...")
    stats = killkrill_service.get_stats()
    print(f"Service stats: {stats}")

    # Wait for flush
    print("\nWaiting for data to flush to KillKrill...")
    time.sleep(5)

    # Final flush
    killkrill_service.flush_logs()
    killkrill_service.flush_metrics()

    # Stop service
    print("Stopping KillKrill service...")
    killkrill_service.stop()

    print("KillKrill integration example completed!")


if __name__ == '__main__':
    main()