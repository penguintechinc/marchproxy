import { describe, it, expect } from 'vitest'

describe('clusterApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../clusterApi')
    expect(mod).toBeDefined()
  })

  it('exports clusterApi object with methods', async () => {
    const {
      clusterApi
    } = await import('../clusterApi')
    
    expect(clusterApi).toBeDefined()
    expect(clusterApi.list).toBeDefined()
    expect(clusterApi.get).toBeDefined()
    expect(clusterApi.create).toBeDefined()
    expect(clusterApi.update).toBeDefined()
    expect(clusterApi.delete).toBeDefined()
  })
})
