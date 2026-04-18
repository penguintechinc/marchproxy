import { describe, it, expect } from 'vitest'

describe('clusters module', () => {
  it('module can be imported', async () => {
    const mod = await import('../clusters')
    expect(mod).toBeDefined()
  })

  it('exports cluster service functions', async () => {
    const { 
      getClusters, getCluster, createCluster, updateCluster, deleteCluster, rotateApiKey
    } = await import('../clusters')
    
    expect(getClusters).toBeDefined()
    expect(getCluster).toBeDefined()
    expect(createCluster).toBeDefined()
    expect(updateCluster).toBeDefined()
    expect(deleteCluster).toBeDefined()
    expect(rotateApiKey).toBeDefined()
  })
})
