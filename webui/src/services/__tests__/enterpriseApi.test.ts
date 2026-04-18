import { describe, it, expect } from 'vitest'

describe('enterpriseApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../enterpriseApi')
    expect(mod).toBeDefined()
  })

  it('exports QoS functions', async () => {
    const { getQoSPolicies, getQoSPolicy, createQoSPolicy, updateQoSPolicy, deleteQoSPolicy } = 
      await import('../enterpriseApi')
    
    expect(getQoSPolicies).toBeDefined()
    expect(getQoSPolicy).toBeDefined()
    expect(createQoSPolicy).toBeDefined()
    expect(updateQoSPolicy).toBeDefined()
    expect(deleteQoSPolicy).toBeDefined()
  })

  it('exports routing functions', async () => {
    const {
      getCloudRoutes, getCloudRoute, createCloudRoute, updateCloudRoute, deleteCloudRoute,
      getBackendHealth, getCloudBackendLocations, getRoutingAlgorithms, updateRoutingAlgorithm
    } = await import('../enterpriseApi')
    
    expect(getCloudRoutes).toBeDefined()
    expect(getCloudRoute).toBeDefined()
    expect(createCloudRoute).toBeDefined()
    expect(updateCloudRoute).toBeDefined()
    expect(deleteCloudRoute).toBeDefined()
    expect(getBackendHealth).toBeDefined()
    expect(getCloudBackendLocations).toBeDefined()
    expect(getRoutingAlgorithms).toBeDefined()
    expect(updateRoutingAlgorithm).toBeDefined()
  })

  it('exports cost analytics functions', async () => {
    const {
      getCostAnalytics, getCostTimeSeries, getCostOptimizations, exportCostReport
    } = await import('../enterpriseApi')
    
    expect(getCostAnalytics).toBeDefined()
    expect(getCostTimeSeries).toBeDefined()
    expect(getCostOptimizations).toBeDefined()
    expect(exportCostReport).toBeDefined()
  })

  it('exports NUMA functions', async () => {
    const {
      getNUMATopology, getNUMAConfig, updateNUMAConfig, getNUMAMetrics, resetNUMAConfig
    } = await import('../enterpriseApi')
    
    expect(getNUMATopology).toBeDefined()
    expect(getNUMAConfig).toBeDefined()
    expect(updateNUMAConfig).toBeDefined()
    expect(getNUMAMetrics).toBeDefined()
    expect(resetNUMAConfig).toBeDefined()
  })
})
