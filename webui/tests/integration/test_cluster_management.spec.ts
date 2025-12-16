/**
 * Integration tests for cluster management.
 */
import { test, expect, Page } from '@playwright/test';

// Helper function to login
async function login(page: Page) {
  await page.goto('/login');
  await page.fill('input[name="email"]', 'admin@test.com');
  await page.fill('input[name="password"]', 'Admin123!');
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/dashboard/);
}

test.describe('Cluster Management', () => {
  test.beforeEach(async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/clusters');
  });

  test('should display clusters list', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Clusters');
    await expect(page.locator('[data-testid="clusters-table"]')).toBeVisible();
  });

  test('should create a new community cluster', async ({ page }) => {
    // Click create button
    await page.click('button:has-text("Create Cluster")');

    // Fill form
    await page.fill('input[name="name"]', 'test-cluster-' + Date.now());
    await page.fill('textarea[name="description"]', 'Test cluster description');
    await page.selectOption('select[name="tier"]', 'community');

    // Submit
    await page.click('button[type="submit"]:has-text("Create")');

    // Should show success message
    await expect(
      page.locator('text=/Cluster created|Success/i')
    ).toBeVisible();

    // Should appear in list
    await expect(
      page.locator('[data-testid="clusters-table"]')
    ).toContainText('test-cluster');
  });

  test('should create an enterprise cluster', async ({ page }) => {
    await page.click('button:has-text("Create Cluster")');

    await page.fill('input[name="name"]', 'enterprise-cluster-' + Date.now());
    await page.fill('textarea[name="description"]', 'Enterprise cluster');
    await page.selectOption('select[name="tier"]', 'enterprise');
    await page.fill('input[name="license_key"]', 'PENG-TEST-TEST-TEST-TEST-ABCD');

    await page.click('button[type="submit"]:has-text("Create")');

    await expect(
      page.locator('text=/Cluster created|Success/i')
    ).toBeVisible();
  });

  test('should show validation errors for invalid input', async ({ page }) => {
    await page.click('button:has-text("Create Cluster")');

    // Submit without filling required fields
    await page.click('button[type="submit"]:has-text("Create")');

    await expect(page.locator('text=/Name is required/i')).toBeVisible();
  });

  test('should edit cluster', async ({ page }) => {
    // Click edit on first cluster
    await page.click('[data-testid="cluster-row"]:first-child button[aria-label="Edit"]');

    // Update description
    await page.fill('textarea[name="description"]', 'Updated description');

    // Save
    await page.click('button[type="submit"]:has-text("Save")');

    await expect(
      page.locator('text=/Cluster updated|Success/i')
    ).toBeVisible();
  });

  test('should view cluster details', async ({ page }) => {
    // Click on first cluster
    await page.click('[data-testid="cluster-row"]:first-child');

    // Should show details
    await expect(page.locator('text=Cluster Details')).toBeVisible();
    await expect(page.locator('text=API Key')).toBeVisible();
    await expect(page.locator('text=Proxies')).toBeVisible();
    await expect(page.locator('text=Services')).toBeVisible();
  });

  test('should regenerate API key', async ({ page }) => {
    await page.click('[data-testid="cluster-row"]:first-child');

    // Get current API key
    const currentKey = await page
      .locator('[data-testid="api-key"]')
      .textContent();

    // Regenerate
    await page.click('button:has-text("Regenerate API Key")');

    // Confirm dialog
    await page.click('button:has-text("Confirm")');

    // Key should change
    await expect(page.locator('[data-testid="api-key"]')).not.toHaveText(
      currentKey || ''
    );

    await expect(
      page.locator('text=/Key regenerated|Success/i')
    ).toBeVisible();
  });

  test('should copy API key to clipboard', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);

    await page.click('[data-testid="cluster-row"]:first-child');

    // Click copy button
    await page.click('button[aria-label="Copy API Key"]');

    // Should show copied message
    await expect(page.locator('text=/Copied/i')).toBeVisible();
  });

  test('should delete cluster', async ({ page }) => {
    // Create a cluster to delete
    await page.click('button:has-text("Create Cluster")');
    const clusterName = 'delete-me-' + Date.now();
    await page.fill('input[name="name"]', clusterName);
    await page.selectOption('select[name="tier"]', 'community');
    await page.click('button[type="submit"]:has-text("Create")');

    await page.waitForTimeout(1000);

    // Find and delete the cluster
    await page.click(`[data-testid="cluster-row"]:has-text("${clusterName}") button[aria-label="Delete"]`);

    // Confirm deletion
    await page.fill('input[name="confirmName"]', clusterName);
    await page.click('button:has-text("Delete")');

    // Should show success and disappear from list
    await expect(
      page.locator('text=/Cluster deleted|Success/i')
    ).toBeVisible();

    await expect(
      page.locator(`text="${clusterName}"`)
    ).not.toBeVisible();
  });

  test('should filter clusters by tier', async ({ page }) => {
    // Select community tier filter
    await page.selectOption('[data-testid="tier-filter"]', 'community');

    // All visible clusters should be community
    const rows = page.locator('[data-testid="cluster-row"]');
    const count = await rows.count();

    for (let i = 0; i < count; i++) {
      await expect(rows.nth(i)).toContainText('Community');
    }
  });

  test('should search clusters by name', async ({ page }) => {
    await page.fill('input[placeholder*="Search"]', 'test');

    // Should filter results
    await expect(
      page.locator('[data-testid="cluster-row"]')
    ).toContainText('test');
  });

  test('should show cluster statistics', async ({ page }) => {
    await page.click('[data-testid="cluster-row"]:first-child');

    // Should show stats
    await expect(page.locator('text=Active Proxies')).toBeVisible();
    await expect(page.locator('text=Total Services')).toBeVisible();
    await expect(page.locator('text=Total Traffic')).toBeVisible();
  });

  test('should paginate clusters list', async ({ page }) => {
    // Assuming there are more than one page of clusters
    const nextButton = page.locator('button[aria-label="Next page"]');

    if (await nextButton.isEnabled()) {
      await nextButton.click();

      // Should show page 2
      await expect(page.locator('text=/Page 2/')).toBeVisible();
    }
  });

  test('should sort clusters by name', async ({ page }) => {
    // Click name column header
    await page.click('[data-testid="column-header-name"]');

    // Should sort ascending
    const firstRow = page.locator('[data-testid="cluster-row"]').first();
    const firstClusterName = await firstRow
      .locator('[data-testid="cluster-name"]')
      .textContent();

    // Click again to sort descending
    await page.click('[data-testid="column-header-name"]');

    const newFirstRow = page.locator('[data-testid="cluster-row"]').first();
    const newFirstClusterName = await newFirstRow
      .locator('[data-testid="cluster-name"]')
      .textContent();

    // Names should be different if there are multiple clusters
    expect(firstClusterName).not.toBe(newFirstClusterName);
  });
});

test.describe('Cluster Access Control', () => {
  test('should restrict cluster operations for non-admin users', async ({ page }) => {
    // Login as regular user
    await page.goto('/login');
    await page.fill('input[name="email"]', 'user@test.com');
    await page.fill('input[name="password"]', 'User123!');
    await page.click('button[type="submit"]');

    await page.goto('/dashboard/clusters');

    // Create button should be disabled or hidden
    const createButton = page.locator('button:has-text("Create Cluster")');
    if (await createButton.isVisible()) {
      await expect(createButton).toBeDisabled();
    }
  });
});
