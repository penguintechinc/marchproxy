import { describe, it, expect } from 'vitest'

describe('serviceApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../serviceApi')
    expect(mod).toBeDefined()
  })

  it('exports serviceApi object with methods', async () => {
    const {
      serviceApi
    } = await import('../serviceApi')
    
    expect(serviceApi).toBeDefined()
    expect(serviceApi.list).toBeDefined()
    expect(serviceApi.get).toBeDefined()
    expect(serviceApi.create).toBeDefined()
    expect(serviceApi.update).toBeDefined()
    expect(serviceApi.delete).toBeDefined()
  })
})
