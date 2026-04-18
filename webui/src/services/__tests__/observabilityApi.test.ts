import { describe, it, expect } from 'vitest'

describe('observabilityApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../observabilityApi')
    expect(mod).toBeDefined()
  })

  it('exports observability functions', async () => {
    const {
      getTraces, getTraceById, getServiceGraph, queryMetrics, queryRangeMetrics,
      getMetricNames, getAlerts, getAlertById, createAlert, updateAlert, deleteAlert,
      getActiveAlerts, acknowledgeAlert, silenceAlert, getJaegerUrl, getServices,
      getOperations, exportTraces, exportMetrics
    } = await import('../observabilityApi')
    
    expect(getTraces).toBeDefined()
    expect(getTraceById).toBeDefined()
    expect(getServiceGraph).toBeDefined()
    expect(queryMetrics).toBeDefined()
    expect(getAlerts).toBeDefined()
  })
})
