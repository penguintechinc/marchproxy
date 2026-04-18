import { describe, it, expect } from 'vitest'

describe('licenseApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../licenseApi')
    expect(mod).toBeDefined()
  })

  it('exports license functions', async () => {
    const {
      getLicenseStatus, checkFeature, getAvailableFeatures, refreshLicenseCache, clearLicenseCache
    } = await import('../licenseApi')
    
    expect(getLicenseStatus).toBeDefined()
    expect(checkFeature).toBeDefined()
    expect(getAvailableFeatures).toBeDefined()
    expect(refreshLicenseCache).toBeDefined()
    expect(clearLicenseCache).toBeDefined()
  })
})
