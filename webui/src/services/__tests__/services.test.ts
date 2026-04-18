import { describe, it, expect } from 'vitest'

describe('services module', () => {
  it('module can be imported', async () => {
    const mod = await import('../services')
    expect(mod).toBeDefined()
  })

  it('exports expected service functions', async () => {
    const { getServices, getService, createService, updateService, deleteService } = 
      await import('../services')
    
    expect(getServices).toBeDefined()
    expect(getService).toBeDefined()
    expect(createService).toBeDefined()
    expect(updateService).toBeDefined()
    expect(deleteService).toBeDefined()
  })
})
