/**
 * Modules API Service
 *
 * API client for Unified NLB Module Management:
 * - Module enable/disable (NLB, ALB, DBLB, AILB, RTMP)
 * - Route configuration per module
 * - Auto-scaling policies
 * - Blue/green deployments
 */

import { apiClient } from './api';
import type {
  Module,
  ModuleRoute,
  AutoScalingPolicy,
  BlueGreenDeployment,
  ModuleMetrics,
  ModuleInstance,
} from './types';

const BASE_URL = '/api/v1/modules';

// Module Management
export const getModules = async (): Promise<Module[]> => {
  const response = await apiClient.get(`${BASE_URL}`);
  return response.data;
};

export const getModule = async (id: number): Promise<Module> => {
  const response = await apiClient.get(`${BASE_URL}/${id}`);
  return response.data;
};

export const enableModule = async (id: number): Promise<Module> => {
  const response = await apiClient.post(`${BASE_URL}/${id}/enable`);
  return response.data;
};

export const disableModule = async (id: number): Promise<Module> => {
  const response = await apiClient.post(`${BASE_URL}/${id}/disable`);
  return response.data;
};

export const getModuleMetrics = async (id: number): Promise<ModuleMetrics> => {
  const response = await apiClient.get(`${BASE_URL}/${id}/metrics`);
  return response.data;
};

export const getModuleInstances = async (id: number): Promise<ModuleInstance[]> => {
  const response = await apiClient.get(`${BASE_URL}/${id}/instances`);
  return response.data;
};

// Module Routes
export const getModuleRoutes = async (moduleId: number): Promise<ModuleRoute[]> => {
  const response = await apiClient.get(`${BASE_URL}/${moduleId}/routes`);
  return response.data;
};

export const getModuleRoute = async (moduleId: number, routeId: number): Promise<ModuleRoute> => {
  const response = await apiClient.get(`${BASE_URL}/${moduleId}/routes/${routeId}`);
  return response.data;
};

export const createModuleRoute = async (
  moduleId: number,
  route: Partial<ModuleRoute>
): Promise<ModuleRoute> => {
  const response = await apiClient.post(`${BASE_URL}/${moduleId}/routes`, route);
  return response.data;
};

export const updateModuleRoute = async (
  moduleId: number,
  routeId: number,
  route: Partial<ModuleRoute>
): Promise<ModuleRoute> => {
  const response = await apiClient.put(`${BASE_URL}/${moduleId}/routes/${routeId}`, route);
  return response.data;
};

export const deleteModuleRoute = async (moduleId: number, routeId: number): Promise<void> => {
  await apiClient.delete(`${BASE_URL}/${moduleId}/routes/${routeId}`);
};

// Auto-Scaling Policies
export const getAutoScalingPolicies = async (
  moduleId: number
): Promise<AutoScalingPolicy[]> => {
  const response = await apiClient.get(`${BASE_URL}/${moduleId}/scaling-policies`);
  return response.data;
};

export const getAutoScalingPolicy = async (
  moduleId: number,
  policyId: number
): Promise<AutoScalingPolicy> => {
  const response = await apiClient.get(`${BASE_URL}/${moduleId}/scaling-policies/${policyId}`);
  return response.data;
};

export const createAutoScalingPolicy = async (
  moduleId: number,
  policy: Partial<AutoScalingPolicy>
): Promise<AutoScalingPolicy> => {
  const response = await apiClient.post(`${BASE_URL}/${moduleId}/scaling-policies`, policy);
  return response.data;
};

export const updateAutoScalingPolicy = async (
  moduleId: number,
  policyId: number,
  policy: Partial<AutoScalingPolicy>
): Promise<AutoScalingPolicy> => {
  const response = await apiClient.put(
    `${BASE_URL}/${moduleId}/scaling-policies/${policyId}`,
    policy
  );
  return response.data;
};

export const deleteAutoScalingPolicy = async (
  moduleId: number,
  policyId: number
): Promise<void> => {
  await apiClient.delete(`${BASE_URL}/${moduleId}/scaling-policies/${policyId}`);
};

export const triggerScaling = async (
  moduleId: number,
  direction: 'up' | 'down',
  count: number = 1
): Promise<void> => {
  await apiClient.post(`${BASE_URL}/${moduleId}/scale`, { direction, count });
};

// Blue/Green Deployments
export const getBlueGreenDeployments = async (
  moduleId: number
): Promise<BlueGreenDeployment[]> => {
  const response = await apiClient.get(`${BASE_URL}/${moduleId}/deployments`);
  return response.data;
};

export const getActiveDeployment = async (
  moduleId: number
): Promise<BlueGreenDeployment | null> => {
  const response = await apiClient.get(`${BASE_URL}/${moduleId}/deployments/active`);
  return response.data;
};

export const createBlueGreenDeployment = async (
  moduleId: number,
  deployment: Partial<BlueGreenDeployment>
): Promise<BlueGreenDeployment> => {
  const response = await apiClient.post(`${BASE_URL}/${moduleId}/deployments`, deployment);
  return response.data;
};

export const updateTrafficWeight = async (
  moduleId: number,
  deploymentId: number,
  blueWeight: number,
  greenWeight: number
): Promise<BlueGreenDeployment> => {
  const response = await apiClient.post(
    `${BASE_URL}/${moduleId}/deployments/${deploymentId}/traffic`,
    {
      traffic_weight_blue: blueWeight,
      traffic_weight_green: greenWeight,
    }
  );
  return response.data;
};

export const rollbackDeployment = async (
  moduleId: number,
  deploymentId: number
): Promise<BlueGreenDeployment> => {
  const response = await apiClient.post(
    `${BASE_URL}/${moduleId}/deployments/${deploymentId}/rollback`
  );
  return response.data;
};

export const finalizeDeployment = async (
  moduleId: number,
  deploymentId: number,
  targetVersion: 'blue' | 'green'
): Promise<BlueGreenDeployment> => {
  const response = await apiClient.post(
    `${BASE_URL}/${moduleId}/deployments/${deploymentId}/finalize`,
    { target_version: targetVersion }
  );
  return response.data;
};
