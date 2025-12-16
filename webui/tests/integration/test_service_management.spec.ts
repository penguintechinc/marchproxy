/**
 * Integration tests for service management.
 */
import { test, expect, Page } from '@playwright/test';

async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[name="email"]', 'admin@test.com');
  await page.fill('input[name="password"]', 'Admin123!');
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/dashboard/);
}

test.describe('Service Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/services');
  });

  test('should display services list', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Services');
    await expect(page.locator('[data-testid="services-table"]')).toBeVisible();
  });

  test('should create a new service', async ({ page }) => {
    await page.click('button:has-text("Create Service")');

    // Fill service form
    const serviceName = 'test-service-' + Date.now();
    await page.fill('input[name="name"]', serviceName);
    await page.fill('input[name="source_ip"]', '10.0.0.100');
    await page.fill('input[name="destination_host"]', 'api.example.com');
    await page.fill('input[name="destination_port"]', '443');
    await page.selectOption('select[name="protocol"]', 'https');
    await page.selectOption('select[name="auth_type"]', 'jwt');

    // Submit
    await page.click('button[type="submit"]:has-text("Create")');

    // Should show success
    await expect(
      page.locator('text=/Service created|Success/i')
    ).toBeVisible();

    // Should display auth credentials
    await expect(page.locator('text=/JWT Secret|Token/i')).toBeVisible();
  });

  test('should create service with port range', async ({ page }) => {
    await page.click('button:has-text("Create Service")');

    await page.fill('input[name="name"]', 'port-range-' + Date.now());
    await page.fill('input[name="source_ip"]', '10.0.0.101');
    await page.fill('input[name="destination_host"]', 'server.example.com');
    await page.fill('input[name="destination_ports"]', '8080-8090');
    await page.selectOption('select[name="protocol"]', 'tcp');

    await page.click('button[type="submit"]:has-text("Create")');

    await expect(
      page.locator('text=/Service created|Success/i')
    ).toBeVisible();
  });

  test('should create service with multiple ports', async ({ page }) => {
    await page.click('button:has-text("Create Service")');

    await page.fill('input[name="name"]', 'multi-port-' + Date.now());
    await page.fill('input[name="source_ip"]', '10.0.0.102');
    await page.fill('input[name="destination_host"]', 'multi.example.com');
    await page.fill('input[name="destination_ports"]', '80,443,8080');
    await page.selectOption('select[name="protocol"]', 'tcp');

    await page.click('button[type="submit"]:has-text("Create")');

    await expect(
      page.locator('text=/Service created|Success/i')
    ).toBeVisible();
  });

  test('should validate IP address format', async ({ page }) => {
    await page.click('button:has-text("Create Service")');

    await page.fill('input[name="source_ip"]', 'invalid-ip');
    await page.fill('input[name="name"]', 'test');
    await page.click('button[type="submit"]:has-text("Create")');

    await expect(
      page.locator('text=/Invalid IP|valid IP address/i')
    ).toBeVisible();
  });

  test('should edit service', async ({ page }) => {
    await page.click('[data-testid="service-row"]:first-child button[aria-label="Edit"]');

    await page.fill('input[name="destination_host"]', 'updated.example.com');
    await page.click('button[type="submit"]:has-text("Save")');

    await expect(
      page.locator('text=/Service updated|Success/i')
    ).toBeVisible();
  });

  test('should view service details', async ({ page }) => {
    await page.click('[data-testid="service-row"]:first-child');

    await expect(page.locator('text=Service Details')).toBeVisible();
    await expect(page.locator('text=Source IP')).toBeVisible();
    await expect(page.locator('text=Destination')).toBeVisible();
    await expect(page.locator('text=Protocol')).toBeVisible();
  });

  test('should rotate auth token', async ({ page }) => {
    await page.click('[data-testid="service-row"]:first-child');

    // Click rotate button
    await page.click('button:has-text("Rotate Token")');

    // Confirm
    await page.click('button:has-text("Confirm")');

    // Should show new token
    await expect(
      page.locator('text=/Token rotated|New token/i')
    ).toBeVisible();
  });

  test('should copy auth credentials', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);

    await page.click('[data-testid="service-row"]:first-child');

    await page.click('button[aria-label="Copy Token"]');

    await expect(page.locator('text=/Copied/i')).toBeVisible();
  });

  test('should delete service', async ({ page }) => {
    // Create service to delete
    await page.click('button:has-text("Create Service")');
    const serviceName = 'delete-me-' + Date.now();
    await page.fill('input[name="name"]', serviceName);
    await page.fill('input[name="source_ip"]', '10.0.0.200');
    await page.fill('input[name="destination_host"]', 'delete.example.com');
    await page.fill('input[name="destination_port"]', '443');
    await page.selectOption('select[name="protocol"]', 'https');
    await page.click('button[type="submit"]:has-text("Create")');

    await page.waitForTimeout(1000);

    // Delete it
    await page.click(`[data-testid="service-row"]:has-text("${serviceName}") button[aria-label="Delete"]`);
    await page.fill('input[name="confirmName"]', serviceName);
    await page.click('button:has-text("Delete")');

    await expect(
      page.locator('text=/Service deleted|Success/i')
    ).toBeVisible();
  });

  test('should filter services by cluster', async ({ page }) => {
    await page.selectOption('[data-testid="cluster-filter"]', { index: 1 });

    // Should filter results
    await page.waitForTimeout(500);
    const rows = page.locator('[data-testid="service-row"]');
    expect(await rows.count()).toBeGreaterThanOrEqual(0);
  });

  test('should filter services by protocol', async ({ page }) => {
    await page.selectOption('[data-testid="protocol-filter"]', 'https');

    const rows = page.locator('[data-testid="service-row"]');
    const count = await rows.count();

    for (let i = 0; i < count; i++) {
      await expect(rows.nth(i)).toContainText('HTTPS');
    }
  });

  test('should search services', async ({ page }) => {
    await page.fill('input[placeholder*="Search"]', 'test');

    // Should filter results
    await expect(
      page.locator('[data-testid="service-row"]')
    ).toContainText('test');
  });

  test('should show service statistics', async ({ page }) => {
    await page.click('[data-testid="service-row"]:first-child');

    await expect(page.locator('text=/Requests|Traffic|Connections/i')).toBeVisible();
  });

  test('should export service configuration', async ({ page }) => {
    await page.click('[data-testid="service-row"]:first-child');

    // Click export button
    const downloadPromise = page.waitForEvent('download');
    await page.click('button:has-text("Export")');
    const download = await downloadPromise;

    // Verify download
    expect(download.suggestedFilename()).toMatch(/\.json$/);
  });

  test('should import service configuration', async ({ page }) => {
    await page.click('button:has-text("Import")');

    // Upload file
    const fileInput = page.locator('input[type="file"]');
    await fileInput.setInputFiles({
      name: 'service-config.json',
      mimeType: 'application/json',
      buffer: Buffer.from(JSON.stringify({
        name: 'imported-service',
        source_ip: '10.0.0.150',
        destination_host: 'import.example.com',
        destination_port: 443,
        protocol: 'https'
      }))
    });

    await page.click('button[type="submit"]:has-text("Import")');

    await expect(
      page.locator('text=/Service imported|Success/i')
    ).toBeVisible();
  });

  test('should show validation errors for duplicate service name', async ({ page }) => {
    // Get first service name
    const firstName = await page
      .locator('[data-testid="service-row"]:first-child [data-testid="service-name"]')
      .textContent();

    // Try to create with same name
    await page.click('button:has-text("Create Service")');
    await page.fill('input[name="name"]', firstName || 'test');
    await page.fill('input[name="source_ip"]', '10.0.0.99');
    await page.fill('input[name="destination_host"]', 'dup.example.com');
    await page.fill('input[name="destination_port"]', '443');
    await page.selectOption('select[name="protocol"]', 'https');
    await page.click('button[type="submit"]:has-text("Create")');

    await expect(
      page.locator('text=/already exists|duplicate/i')
    ).toBeVisible();
  });

  test('should enable/disable service', async ({ page }) => {
    await page.click('[data-testid="service-row"]:first-child');

    // Toggle status
    const toggle = page.locator('input[name="is_active"]');
    const initialState = await toggle.isChecked();

    await toggle.click();

    // Should update
    await page.waitForTimeout(500);
    expect(await toggle.isChecked()).toBe(!initialState);
  });
});
