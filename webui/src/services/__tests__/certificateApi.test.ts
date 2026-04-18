import { describe, it, expect } from 'vitest'

describe('certificateApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../certificateApi')
    expect(mod).toBeDefined()
  })

  it('exports certificateApi object with methods', async () => {
    const {
      certificateApi
    } = await import('../certificateApi')
    
    expect(certificateApi).toBeDefined()
    expect(certificateApi.list).toBeDefined()
    expect(certificateApi.get).toBeDefined()
    expect(certificateApi.upload).toBeDefined()
    expect(certificateApi.update).toBeDefined()
    expect(certificateApi.delete).toBeDefined()
  })
})
