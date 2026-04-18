import { describe, it, expect } from 'vitest'

describe('securityApi module', () => {
  it('module can be imported', async () => {
    const mod = await import('../securityApi')
    expect(mod).toBeDefined()
  })

  it('exports security functions', async () => {
    const {
      getPolicies, getPolicy, savePolicy, deletePolicy, getPolicyVersions,
      testPolicy, getPolicyTemplates, validatePolicy, getAuditLogs, exportAuditLogs,
      verifyAuditLogIntegrity, getComplianceStatus, runComplianceCheck,
      uploadComplianceEvidence, generateComplianceReport, getCertificates,
      getCertificate, uploadCertificate, deleteCertificate, revokeCertificate,
      validateCertificate, getCRL, updateCertificateRotation, getExpiringCertificates
    } = await import('../securityApi')
    
    expect(getPolicies).toBeDefined()
    expect(getPolicy).toBeDefined()
    expect(savePolicy).toBeDefined()
    expect(getAuditLogs).toBeDefined()
    expect(getComplianceStatus).toBeDefined()
  })
})
