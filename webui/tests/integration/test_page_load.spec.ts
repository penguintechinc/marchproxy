/**
 * Integration tests for page loading and rendering.
 * Tests that all pages load without JavaScript errors and render main content.
 */
import { test, expect, Page } from '@playwright/test';

/**
 * Helper to login before testing protected pages
 */
async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[name="email"]', 'admin@test.com');
  await page.fill('input[name="password"]', 'Admin123!');
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/dashboard/);
}

/**
 * Helper to check for JavaScript errors on the page
 */
function setupErrorTracking(page: Page): Promise<Error[]> {
  const errors: Error[] = [];
  page.on('pageerror', (error) => {
    errors.push(error);
  });
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(new Error(`Console error: ${msg.text()}`));
    }
  });
  return Promise.resolve(errors);
}

test.describe('Page Load - Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load dashboard without JavaScript errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
    expect(page.url()).toContain('/dashboard');
  });

  test('should display dashboard main content', async ({ page }) => {
    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    // Check for main dashboard content
    await expect(page.locator('h1')).toContainText('Dashboard');

    // Check for stat cards
    await expect(page.locator('text=/Clusters|Services|Proxies|Active/i')).toBeVisible();
  });

  test('should load without console errors', async ({ page }) => {
    const consoleMessages: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleMessages.push(`${msg.type()}: ${msg.text()}`);
      }
    });

    await page.goto('/dashboard');
    await page.waitForLoadState('networkidle');

    expect(consoleMessages).toHaveLength(0);
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Services', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load services page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/services');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display services page content', async ({ page }) => {
    await page.goto('/dashboard/services');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Services');
    await expect(page.locator('[data-testid="services-table"]')).toBeVisible();
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard/services');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Clusters', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load clusters page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/clusters');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display clusters page content', async ({ page }) => {
    await page.goto('/dashboard/clusters');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Clusters');
    await expect(page.locator('[data-testid="clusters-table"]')).toBeVisible();
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard/clusters');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Proxies', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load proxies page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/proxies');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display proxies page content', async ({ page }) => {
    await page.goto('/dashboard/proxies');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Proxies');
    await expect(page.locator('[data-testid="proxies-table"]')).toBeVisible();
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard/proxies');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Certificates', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load certificates page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/certificates');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display certificates page content', async ({ page }) => {
    await page.goto('/dashboard/certificates');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Certificates');
    await expect(page.locator('[data-testid="certificates-table"]')).toBeVisible();
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard/certificates');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Settings', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load settings page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/settings');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display settings page content', async ({ page }) => {
    await page.goto('/dashboard/settings');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Settings');
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard/settings');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Modules', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load module manager without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/modules/manager');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display module manager content', async ({ page }) => {
    await page.goto('/dashboard/modules/manager');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Module Manager');
  });

  test('should load module routes without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/modules/routes');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display module routes content', async ({ page }) => {
    await page.goto('/dashboard/modules/routes');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Route Editor');
  });

  test('should return 200 for both module pages', async ({ page }) => {
    let response = await page.goto('/dashboard/modules/manager');
    expect(response?.status()).toBe(200);

    response = await page.goto('/dashboard/modules/routes');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Scaling', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load auto scaling page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/scaling/auto');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display auto scaling content', async ({ page }) => {
    await page.goto('/dashboard/scaling/auto');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Auto Scaling');
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard/scaling/auto');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Deployments', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load blue-green deployment page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/deployments/blue-green');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display blue-green deployment content', async ({ page }) => {
    await page.goto('/dashboard/deployments/blue-green');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Blue-Green Deployment');
  });

  test('should return 200 status code', async ({ page }) => {
    const response = await page.goto('/dashboard/deployments/blue-green');
    expect(response?.status()).toBe(200);
  });
});

test.describe('Page Load - Security', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load mTLS page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/security/mtls');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display mTLS content', async ({ page }) => {
    await page.goto('/dashboard/security/mtls');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('mTLS Configuration');
  });

  test('should load audit logs page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/security/audit-logs');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display audit logs content', async ({ page }) => {
    await page.goto('/dashboard/security/audit-logs');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Audit Logs');
  });

  test('should load compliance page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/security/compliance');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display compliance content', async ({ page }) => {
    await page.goto('/dashboard/security/compliance');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Compliance');
  });

  test('should load policy editor page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/security/policy-editor');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display policy editor content', async ({ page }) => {
    await page.goto('/dashboard/security/policy-editor');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Policy Editor');
  });

  test('should load policy tester page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/security/policy-tester');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display policy tester content', async ({ page }) => {
    await page.goto('/dashboard/security/policy-tester');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Policy Tester');
  });

  test('should return 200 for all security pages', async ({ page }) => {
    const securityPages = [
      '/dashboard/security/mtls',
      '/dashboard/security/audit-logs',
      '/dashboard/security/compliance',
      '/dashboard/security/policy-editor',
      '/dashboard/security/policy-tester',
    ];

    for (const path of securityPages) {
      const response = await page.goto(path);
      expect(response?.status()).toBe(200);
    }
  });
});

test.describe('Page Load - Enterprise', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load multi-cloud page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/enterprise/multi-cloud');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display multi-cloud content', async ({ page }) => {
    await page.goto('/dashboard/enterprise/multi-cloud');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Multi-Cloud Routing');
  });

  test('should load NUMA page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/enterprise/numa');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display NUMA content', async ({ page }) => {
    await page.goto('/dashboard/enterprise/numa');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('NUMA Configuration');
  });

  test('should load traffic shaping page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/enterprise/traffic-shaping');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display traffic shaping content', async ({ page }) => {
    await page.goto('/dashboard/enterprise/traffic-shaping');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Traffic Shaping');
  });

  test('should load cost analytics page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/enterprise/cost-analytics');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display cost analytics content', async ({ page }) => {
    await page.goto('/dashboard/enterprise/cost-analytics');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Cost Analytics');
  });

  test('should return 200 for all enterprise pages', async ({ page }) => {
    const enterprisePages = [
      '/dashboard/enterprise/multi-cloud',
      '/dashboard/enterprise/numa',
      '/dashboard/enterprise/traffic-shaping',
      '/dashboard/enterprise/cost-analytics',
    ];

    for (const path of enterprisePages) {
      const response = await page.goto(path);
      expect(response?.status()).toBe(200);
    }
  });
});

test.describe('Page Load - Observability', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
  });

  test('should load metrics page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/observability/metrics');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display metrics page content', async ({ page }) => {
    await page.goto('/dashboard/observability/metrics');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Metrics');
  });

  test('should load tracing page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/observability/tracing');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display tracing page content', async ({ page }) => {
    await page.goto('/dashboard/observability/tracing');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Tracing');
  });

  test('should load alerts page without errors', async ({ page }) => {
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));

    await page.goto('/dashboard/observability/alerts');
    await page.waitForLoadState('networkidle');

    expect(errors).toHaveLength(0);
  });

  test('should display alerts page content', async ({ page }) => {
    await page.goto('/dashboard/observability/alerts');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('h1')).toContainText('Alerts');
  });

  test('should return 200 for all observability pages', async ({ page }) => {
    const observabilityPages = [
      '/dashboard/observability/metrics',
      '/dashboard/observability/tracing',
      '/dashboard/observability/alerts',
    ];

    for (const path of observabilityPages) {
      const response = await page.goto(path);
      expect(response?.status()).toBe(200);
    }
  });
});
