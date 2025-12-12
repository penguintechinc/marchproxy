/**
 * React hook for Multi-Cloud Routing API
 *
 * Manages route tables, health monitoring, and cost analytics.
 */

import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '../services/api';

export type CloudProvider = 'aws' | 'gcp' | 'azure' | 'on_prem';
export type RoutingAlgorithm = 'latency' | 'cost' | 'geo' | 'weighted_rr' | 'failover';
export type HealthCheckProtocol = 'tcp' | 'http' | 'https' | 'icmp';

export interface HealthProbeConfig {
  protocol: HealthCheckProtocol;
  port?: number;
  path?: string;
  interval_seconds: number;
  timeout_seconds: number;
  unhealthy_threshold: number;
  healthy_threshold: number;
}

export interface CloudRoute {
  provider: CloudProvider;
  region: string;
  endpoint: string;
  weight: number;
  cost_per_gb?: number;
  is_active: boolean;
}

export interface RouteHealthStatus {
  endpoint: string;
  is_healthy: boolean;
  last_check: string;
  rtt_ms?: number;
  consecutive_failures: number;
  consecutive_successes: number;
}

export interface RouteTable {
  id: number;
  name: string;
  description?: string;
  service_id: number;
  cluster_id: number;
  algorithm: RoutingAlgorithm;
  routes: CloudRoute[];
  health_probe: HealthProbeConfig;
  enable_auto_failover: boolean;
  enabled: boolean;
  created_at: string;
  updated_at: string;
  health_status?: RouteHealthStatus[];
}

export interface RouteTableCreate {
  name: string;
  description?: string;
  service_id: number;
  cluster_id: number;
  algorithm: RoutingAlgorithm;
  routes: CloudRoute[];
  health_probe: HealthProbeConfig;
  enable_auto_failover?: boolean;
  enabled?: boolean;
}

export interface RouteTableUpdate {
  name?: string;
  description?: string;
  algorithm?: RoutingAlgorithm;
  routes?: CloudRoute[];
  health_probe?: HealthProbeConfig;
  enable_auto_failover?: boolean;
  enabled?: boolean;
}

export interface UseMultiCloudReturn {
  routeTables: RouteTable[];
  loading: boolean;
  error: string | null;
  hasAccess: boolean;
  fetchRouteTables: (clusterId?: number, serviceId?: number) => Promise<void>;
  createRouteTable: (routeTable: RouteTableCreate) => Promise<RouteTable | null>;
  updateRouteTable: (id: number, routeTable: RouteTableUpdate) => Promise<RouteTable | null>;
  deleteRouteTable: (id: number) => Promise<boolean>;
  getRouteHealth: (id: number) => Promise<RouteHealthStatus[] | null>;
  enableRouteTable: (id: number) => Promise<boolean>;
  disableRouteTable: (id: number) => Promise<boolean>;
  testFailover: (id: number, endpoint?: string) => Promise<any>;
  getCostAnalytics: (params?: any) => Promise<any>;
}

export const useMultiCloud = (): UseMultiCloudReturn => {
  const [routeTables, setRouteTables] = useState<RouteTable[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [hasAccess, setHasAccess] = useState<boolean>(true);

  const fetchRouteTables = useCallback(async (
    clusterId?: number,
    serviceId?: number
  ) => {
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams();
      if (clusterId) params.append('cluster_id', clusterId.toString());
      if (serviceId) params.append('service_id', serviceId.toString());

      const response = await apiClient.get(
        `/api/v1/multi-cloud/routes?${params.toString()}`
      );

      setRouteTables(response.data);
      setHasAccess(true);
    } catch (err: any) {
      if (err.response?.status === 403) {
        setHasAccess(false);
        setError('Enterprise feature not available. Please upgrade your license.');
      } else {
        setError(err.message || 'Failed to fetch route tables');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  const createRouteTable = useCallback(async (
    routeTable: RouteTableCreate
  ): Promise<RouteTable | null> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        '/api/v1/multi-cloud/routes',
        routeTable
      );

      const newRouteTable = response.data;
      setRouteTables(prev => [...prev, newRouteTable]);
      return newRouteTable;
    } catch (err: any) {
      if (err.response?.status === 403) {
        setHasAccess(false);
        setError('Enterprise feature not available. Please upgrade your license.');
      } else {
        setError(err.message || 'Failed to create route table');
      }
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const updateRouteTable = useCallback(async (
    id: number,
    routeTable: RouteTableUpdate
  ): Promise<RouteTable | null> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.put(
        `/api/v1/multi-cloud/routes/${id}`,
        routeTable
      );

      const updatedRouteTable = response.data;
      setRouteTables(prev =>
        prev.map(rt => rt.id === id ? updatedRouteTable : rt)
      );
      return updatedRouteTable;
    } catch (err: any) {
      setError(err.message || 'Failed to update route table');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const deleteRouteTable = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      await apiClient.delete(`/api/v1/multi-cloud/routes/${id}`);
      setRouteTables(prev => prev.filter(rt => rt.id !== id));
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to delete route table');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const getRouteHealth = useCallback(async (
    id: number
  ): Promise<RouteHealthStatus[] | null> => {
    try {
      const response = await apiClient.get(
        `/api/v1/multi-cloud/routes/${id}/health`
      );
      return response.data;
    } catch (err: any) {
      setError(err.message || 'Failed to fetch route health');
      return null;
    }
  }, []);

  const enableRouteTable = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        `/api/v1/multi-cloud/routes/${id}/enable`
      );

      const updatedRouteTable = response.data;
      setRouteTables(prev =>
        prev.map(rt => rt.id === id ? updatedRouteTable : rt)
      );
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to enable route table');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const disableRouteTable = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        `/api/v1/multi-cloud/routes/${id}/disable`
      );

      const updatedRouteTable = response.data;
      setRouteTables(prev =>
        prev.map(rt => rt.id === id ? updatedRouteTable : rt)
      );
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to disable route table');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const testFailover = useCallback(async (
    id: number,
    endpoint?: string
  ): Promise<any> => {
    try {
      const params = endpoint ? `?simulate_failure_endpoint=${endpoint}` : '';
      const response = await apiClient.post(
        `/api/v1/multi-cloud/routes/${id}/test-failover${params}`
      );
      return response.data;
    } catch (err: any) {
      setError(err.message || 'Failed to test failover');
      return null;
    }
  }, []);

  const getCostAnalytics = useCallback(async (params?: any): Promise<any> => {
    try {
      const queryParams = new URLSearchParams(params).toString();
      const response = await apiClient.get(
        `/api/v1/multi-cloud/analytics/cost?${queryParams}`
      );
      return response.data;
    } catch (err: any) {
      setError(err.message || 'Failed to fetch cost analytics');
      return null;
    }
  }, []);

  return {
    routeTables,
    loading,
    error,
    hasAccess,
    fetchRouteTables,
    createRouteTable,
    updateRouteTable,
    deleteRouteTable,
    getRouteHealth,
    enableRouteTable,
    disableRouteTable,
    testFailover,
    getCostAnalytics
  };
};
