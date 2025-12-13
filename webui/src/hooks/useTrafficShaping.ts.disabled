/**
 * React hook for Traffic Shaping API
 *
 * Manages QoS policies and traffic shaping configuration.
 */

import { useState, useEffect, useCallback } from 'react';
import { apiClient } from '../services/api';

export interface BandwidthLimit {
  ingress_mbps?: number;
  egress_mbps?: number;
  burst_size_kb?: number;
}

export interface PriorityQueueConfig {
  priority: 'P0' | 'P1' | 'P2' | 'P3';
  weight: number;
  max_latency_ms?: number;
  dscp_marking: 'EF' | 'AF41' | 'AF31' | 'AF21' | 'AF11' | 'BE';
}

export interface QoSPolicy {
  id: number;
  name: string;
  description?: string;
  service_id: number;
  cluster_id: number;
  bandwidth: BandwidthLimit;
  priority_config: PriorityQueueConfig;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface QoSPolicyCreate {
  name: string;
  description?: string;
  service_id: number;
  cluster_id: number;
  bandwidth: BandwidthLimit;
  priority_config: PriorityQueueConfig;
  enabled?: boolean;
}

export interface QoSPolicyUpdate {
  name?: string;
  description?: string;
  bandwidth?: BandwidthLimit;
  priority_config?: PriorityQueueConfig;
  enabled?: boolean;
}

export interface UseTrafficShapingReturn {
  policies: QoSPolicy[];
  loading: boolean;
  error: string | null;
  hasAccess: boolean;
  fetchPolicies: (clusterId?: number, serviceId?: number) => Promise<void>;
  createPolicy: (policy: QoSPolicyCreate) => Promise<QoSPolicy | null>;
  updatePolicy: (id: number, policy: QoSPolicyUpdate) => Promise<QoSPolicy | null>;
  deletePolicy: (id: number) => Promise<boolean>;
  enablePolicy: (id: number) => Promise<boolean>;
  disablePolicy: (id: number) => Promise<boolean>;
}

export const useTrafficShaping = (): UseTrafficShapingReturn => {
  const [policies, setPolicies] = useState<QoSPolicy[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [hasAccess, setHasAccess] = useState<boolean>(true);

  const fetchPolicies = useCallback(async (
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
        `/api/v1/traffic-shaping/policies?${params.toString()}`
      );

      setPolicies(response.data);
      setHasAccess(true);
    } catch (err: any) {
      if (err.response?.status === 403) {
        setHasAccess(false);
        setError('Enterprise feature not available. Please upgrade your license.');
      } else {
        setError(err.message || 'Failed to fetch QoS policies');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  const createPolicy = useCallback(async (
    policy: QoSPolicyCreate
  ): Promise<QoSPolicy | null> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        '/api/v1/traffic-shaping/policies',
        policy
      );

      const newPolicy = response.data;
      setPolicies(prev => [...prev, newPolicy]);
      return newPolicy;
    } catch (err: any) {
      if (err.response?.status === 403) {
        setHasAccess(false);
        setError('Enterprise feature not available. Please upgrade your license.');
      } else {
        setError(err.message || 'Failed to create QoS policy');
      }
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const updatePolicy = useCallback(async (
    id: number,
    policy: QoSPolicyUpdate
  ): Promise<QoSPolicy | null> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.put(
        `/api/v1/traffic-shaping/policies/${id}`,
        policy
      );

      const updatedPolicy = response.data;
      setPolicies(prev =>
        prev.map(p => p.id === id ? updatedPolicy : p)
      );
      return updatedPolicy;
    } catch (err: any) {
      setError(err.message || 'Failed to update QoS policy');
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  const deletePolicy = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      await apiClient.delete(`/api/v1/traffic-shaping/policies/${id}`);
      setPolicies(prev => prev.filter(p => p.id !== id));
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to delete QoS policy');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const enablePolicy = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        `/api/v1/traffic-shaping/policies/${id}/enable`
      );

      const updatedPolicy = response.data;
      setPolicies(prev =>
        prev.map(p => p.id === id ? updatedPolicy : p)
      );
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to enable QoS policy');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  const disablePolicy = useCallback(async (id: number): Promise<boolean> => {
    setLoading(true);
    setError(null);

    try {
      const response = await apiClient.post(
        `/api/v1/traffic-shaping/policies/${id}/disable`
      );

      const updatedPolicy = response.data;
      setPolicies(prev =>
        prev.map(p => p.id === id ? updatedPolicy : p)
      );
      return true;
    } catch (err: any) {
      setError(err.message || 'Failed to disable QoS policy');
      return false;
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    policies,
    loading,
    error,
    hasAccess,
    fetchPolicies,
    createPolicy,
    updatePolicy,
    deletePolicy,
    enablePolicy,
    disablePolicy
  };
};
