import { describe, it, expect } from 'vitest'

describe('modulesApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../modulesApi')
    expect(mod).toBeDefined()
  })

  it('exports module functions', async () => {
    const {
      getModules, getModule, enableModule, disableModule,
      getModuleMetrics, getModuleInstances, getModuleRoutes, updateModuleRoute,
      getAutoScalingPolicy, updateAutoScalingPolicy, getBlueGreenDeployment,
      createBlueGreenDeployment, promoteBlueGreenDeployment
    } = await import('../modulesApi')
    
    expect(getModules).toBeDefined()
    expect(getModule).toBeDefined()
    expect(enableModule).toBeDefined()
    expect(disableModule).toBeDefined()
    expect(getModuleMetrics).toBeDefined()
    expect(getModuleInstances).toBeDefined()
    expect(getModuleRoutes).toBeDefined()
  })
})
