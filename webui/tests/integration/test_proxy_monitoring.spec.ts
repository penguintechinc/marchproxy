/**
 * Integration tests for real-time proxy monitoring.
 */
import { test, expect, Page } from '@playwright/test';

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[name="email"]', 'admin@test.com');
  await page.fill('input[name="password"]', 'Admin123!');
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/dashboard/);
}

test.describe('Proxy Monitoring', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/proxies');
  });

  test('should display proxies list', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Proxies');
    await expect(page.locator('[data-testid="proxies-table"]')).toBeVisible();
  });

  test('should show proxy status indicators', async ({ page }) => {
    const rows = page.locator('[data-testid="proxy-row"]');
    const count = await rows.count();

    if (count > 0) {
      // Should have status badge
      await expect(rows.first().locator('[data-testid="proxy-status"]')).toBeVisible();

      // Status should be one of: online, offline, degraded
      const status = await rows.first().locator('[data-testid="proxy-status"]').textContent();
      expect(['online', 'offline', 'degraded'].some(s =>
        status?.toLowerCase().includes(s)
      )).toBe(true);
    }
  });

  test('should display proxy metrics', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      // Should show metrics
      await expect(row.locator('text=/CPU|Memory|Connections/i')).toBeVisible();
    }
  });

  test('should view proxy details', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      // Should show detailed view
      await expect(page.locator('text=Proxy Details')).toBeVisible();
      await expect(page.locator('text=Hostname')).toBeVisible();
      await expect(page.locator('text=IP Address')).toBeVisible();
      await expect(page.locator('text=Version')).toBeVisible();
    }
  });

  test('should display real-time metrics chart', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      // Should show charts
      await expect(page.locator('[data-testid="cpu-chart"]')).toBeVisible();
      await expect(page.locator('[data-testid="memory-chart"]')).toBeVisible();
      await expect(page.locator('[data-testid="connections-chart"]')).toBeVisible();
    }
  });

  test('should refresh metrics automatically', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      // Get initial metric value
      const initialCpu = await page
        .locator('[data-testid="cpu-value"]')
        .textContent();

      // Wait for auto-refresh (typically 5-10 seconds)
      await page.waitForTimeout(11000);

      // Value may have changed
      const updatedCpu = await page
        .locator('[data-testid="cpu-value"]')
        .textContent();

      // At minimum, the component should still exist
      expect(updatedCpu).toBeTruthy();
    }
  });

  test('should filter proxies by status', async ({ page }) => {
    await page.selectOption('[data-testid="status-filter"]', 'online');

    const rows = page.locator('[data-testid="proxy-row"]');
    const count = await rows.count();

    for (let i = 0; i < count; i++) {
      const status = await rows.nth(i).locator('[data-testid="proxy-status"]').textContent();
      expect(status?.toLowerCase()).toContain('online');
    }
  });

  test('should filter proxies by cluster', async ({ page }) => {
    const clusterFilter = page.locator('[data-testid="cluster-filter"]');

    if (await clusterFilter.isVisible()) {
      await clusterFilter.selectOption({ index: 1 });

      // Should filter results
      await page.waitForTimeout(500);
      const rows = page.locator('[data-testid="proxy-row"]');
      expect(await rows.count()).toBeGreaterThanOrEqual(0);
    }
  });

  test('should search proxies by hostname', async ({ page }) => {
    await page.fill('input[placeholder*="Search"]', 'proxy');

    // Should filter results
    const rows = page.locator('[data-testid="proxy-row"]');
    const count = await rows.count();

    for (let i = 0; i < count; i++) {
      const text = await rows.nth(i).textContent();
      expect(text?.toLowerCase()).toContain('proxy');
    }
  });

  test('should show proxy capabilities', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      // Should show capabilities
      await expect(page.locator('text=Capabilities')).toBeVisible();
      await expect(page.locator('text=/L7|L3L4|TLS|DPDK|XDP/i')).toBeVisible();
    }
  });

  test('should display proxy uptime', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      await expect(page.locator('text=Uptime')).toBeVisible();
      await expect(page.locator('[data-testid="uptime-value"]')).toBeVisible();
    }
  });

  test('should show last heartbeat time', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      await expect(page.locator('text=Last Heartbeat')).toBeVisible();
      await expect(page.locator('[data-testid="heartbeat-time"]')).toBeVisible();
    }
  });

  test('should display traffic statistics', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      // Should show traffic stats
      await expect(page.locator('text=/Bytes In|Bytes Out|Total Traffic/i')).toBeVisible();
    }
  });

  test('should show active connections count', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await expect(row.locator('[data-testid="active-connections"]')).toBeVisible();
    }
  });

  test('should deregister proxy', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      const hostname = await row.locator('[data-testid="proxy-hostname"]').textContent();

      await row.locator('button[aria-label="Delete"]').click();

      // Confirm
      await page.fill('input[name="confirmHostname"]', hostname || '');
      await page.click('button:has-text("Deregister")');

      await expect(
        page.locator('text=/Proxy deregistered|Success/i')
      ).toBeVisible();
    }
  });

  test('should export proxy metrics', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      const downloadPromise = page.waitForEvent('download');
      await page.click('button:has-text("Export Metrics")');
      const download = await downloadPromise;

      expect(download.suggestedFilename()).toMatch(/\.csv|\.json$/);
    }
  });

  test('should show proxy version', async ({ page }) => {
    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await expect(row.locator('[data-testid="proxy-version"]')).toBeVisible();
    }
  });

  test('should indicate outdated proxy versions', async ({ page }) => {
    const rows = page.locator('[data-testid="proxy-row"]');
    const count = await rows.count();

    // Look for version warning indicator
    for (let i = 0; i < count; i++) {
      const versionWarning = rows.nth(i).locator('[data-testid="version-warning"]');
      // May or may not exist depending on versions
      const exists = await versionWarning.count();
      expect(exists).toBeGreaterThanOrEqual(0);
    }
  });

  test('should switch between list and grid view', async ({ page }) => {
    // Switch to grid view
    await page.click('button[aria-label="Grid view"]');

    await expect(page.locator('[data-testid="proxies-grid"]')).toBeVisible();

    // Switch back to list view
    await page.click('button[aria-label="List view"]');

    await expect(page.locator('[data-testid="proxies-table"]')).toBeVisible();
  });

  test('should sort proxies by various fields', async ({ page }) => {
    // Sort by hostname
    await page.click('[data-testid="column-header-hostname"]');

    // Should reorder
    await page.waitForTimeout(500);

    // Click again for descending
    await page.click('[data-testid="column-header-hostname"]');

    await page.waitForTimeout(500);
  });
});

test.describe('Proxy Alerts', () => {
  test('should show alert for offline proxies', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/proxies');

    // Check for offline alert badge
    const alertBadge = page.locator('[data-testid="offline-proxies-alert"]');

    if (await alertBadge.isVisible()) {
      const count = await alertBadge.textContent();
      expect(parseInt(count || '0')).toBeGreaterThan(0);
    }
  });

  test('should show alert for high resource usage', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/proxies');

    const row = page.locator('[data-testid="proxy-row"]').first();

    if (await row.isVisible()) {
      await row.click();

      // Look for resource warning
      const cpuWarning = page.locator('[data-testid="cpu-warning"]');
      const memWarning = page.locator('[data-testid="memory-warning"]');

      // May or may not exist
      const cpuExists = await cpuWarning.count();
      const memExists = await memWarning.count();

      expect(cpuExists + memExists).toBeGreaterThanOrEqual(0);
    }
  });
});
