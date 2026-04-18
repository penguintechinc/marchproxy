import { describe, it, expect } from 'vitest'

describe('proxyApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../proxyApi')
    expect(mod).toBeDefined()
  })

  it('exports proxyApi object with methods', async () => {
    const {
      proxyApi
    } = await import('../proxyApi')
    
    expect(proxyApi).toBeDefined()
    expect(proxyApi.list).toBeDefined()
    expect(proxyApi.get).toBeDefined()
    expect(proxyApi.deregister).toBeDefined()
    expect(proxyApi.getMetrics).toBeDefined()
  })
})
