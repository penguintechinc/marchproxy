/**
 * Integration tests for login flow with 2FA.
 */
import { test, expect, Page } from '@playwright/test';

test.describe('Login Flow', () => {
  let page: Page;

  test.beforeEach(async ({ page: testPage }) => {
    page = testPage;
    await page.goto('/login');
  });

  test('should display login form', async () => {
    await expect(page.locator('h1')).toContainText('Login');
    await expect(page.locator('input[name="email"]')).toBeVisible();
    await expect(page.locator('input[name="password"]')).toBeVisible();
    await expect(page.locator('button[type="submit"]')).toBeVisible();
  });

  test('should show validation errors for empty fields', async () => {
    await page.click('button[type="submit"]');

    await expect(page.locator('text=Email is required')).toBeVisible();
    await expect(page.locator('text=Password is required')).toBeVisible();
  });

  test('should show error for invalid credentials', async () => {
    await page.fill('input[name="email"]', 'invalid@test.com');
    await page.fill('input[name="password"]', 'wrongpassword');
    await page.click('button[type="submit"]');

    await expect(
      page.locator('text=/Invalid credentials|incorrect/i')
    ).toBeVisible();
  });

  test('should successfully login with valid credentials', async () => {
    // Use test credentials
    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.click('button[type="submit"]');

    // Should redirect to dashboard
    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.locator('text=Dashboard')).toBeVisible();
  });

  test('should handle 2FA flow', async () => {
    // Login with 2FA-enabled account
    await page.fill('input[name="email"]', '2fa@test.com');
    await page.fill('input[name="password"]', 'Test123!');
    await page.click('button[type="submit"]');

    // Should show 2FA input
    await expect(page.locator('text=Two-Factor Authentication')).toBeVisible();
    await expect(page.locator('input[name="code"]')).toBeVisible();

    // Enter 2FA code
    await page.fill('input[name="code"]', '123456');
    await page.click('button[type="submit"]');

    // Should complete login or show error for invalid code
    const dashboard = page.locator('text=Dashboard');
    const error = page.locator('text=/Invalid code|incorrect/i');

    await expect(
      dashboard.or(error)
    ).toBeVisible();
  });

  test('should persist login after page refresh', async () => {
    // Login
    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/dashboard/);

    // Refresh page
    await page.reload();

    // Should still be logged in
    await expect(page).toHaveURL(/\/dashboard/);
    await expect(page.locator('text=Dashboard')).toBeVisible();
  });

  test('should logout successfully', async () => {
    // Login first
    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/dashboard/);

    // Click logout
    await page.click('button[aria-label="Account"]');
    await page.click('text=Logout');

    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
  });

  test('should show password reset link', async () => {
    await expect(page.locator('a[href="/forgot-password"]')).toBeVisible();
  });

  test('should toggle password visibility', async () => {
    const passwordInput = page.locator('input[name="password"]');
    const toggleButton = page.locator('button[aria-label="Toggle password visibility"]');

    // Initially hidden
    await expect(passwordInput).toHaveAttribute('type', 'password');

    // Click to show
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'text');

    // Click to hide again
    await toggleButton.click();
    await expect(passwordInput).toHaveAttribute('type', 'password');
  });

  test('should validate email format', async () => {
    await page.fill('input[name="email"]', 'invalid-email');
    await page.fill('input[name="password"]', 'Test123!');
    await page.click('button[type="submit"]');

    await expect(
      page.locator('text=/Invalid email|valid email/i')
    ).toBeVisible();
  });

  test('should handle session expiry', async () => {
    // Login
    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/dashboard/);

    // Clear session storage to simulate expiry
    await page.evaluate(() => {
      localStorage.clear();
      sessionStorage.clear();
    });

    // Navigate to protected route
    await page.goto('/dashboard/clusters');

    // Should redirect to login
    await expect(page).toHaveURL(/\/login/);
  });
});

test.describe('Remember Me', () => {
  test('should save credentials when remember me is checked', async ({ page }) => {
    await page.goto('/login');

    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.check('input[name="remember"]');
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/dashboard/);

    // Close and reopen (simulated by clearing session storage only)
    await page.evaluate(() => sessionStorage.clear());
    await page.reload();

    // Should still be logged in via localStorage
    await expect(page).toHaveURL(/\/dashboard/);
  });
});
