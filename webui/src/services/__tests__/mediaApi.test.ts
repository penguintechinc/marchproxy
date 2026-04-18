import { describe, it, expect } from 'vitest'

describe('mediaApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../mediaApi')
    expect(mod).toBeDefined()
  })

  it('exports media configuration functions', async () => {
    const {
      getMediaConfig, updateMediaConfig, getActiveStreams, getStream, stopStream,
      getCapabilities, getMediaStats
    } = await import('../mediaApi')
    
    expect(getMediaConfig).toBeDefined()
    expect(updateMediaConfig).toBeDefined()
    expect(getActiveStreams).toBeDefined()
    expect(getStream).toBeDefined()
    expect(stopStream).toBeDefined()
    expect(getCapabilities).toBeDefined()
    expect(getMediaStats).toBeDefined()
  })
})
