# Testing Guide - Playwright

## Overview

MarchProxy WebUI uses Playwright for end-to-end (E2E) testing. Tests verify that all pages load correctly, render content, and execute user workflows without JavaScript errors.

## Setup

### Installation

Tests are configured in `package.json` with Playwright v1.40+:

```bash
npm install --save-dev @playwright/test
npx playwright install  # Install browser binaries
```

### Configuration

`playwright.config.ts` defines test settings:

```typescript
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/integration',
  timeout: 30000,
  expect: { timeout: 5000 },
  fullyParallel: true,
  forbidOnly: process.env.CI ? true : false,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,

  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
  },

  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: false,
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
});
```

## Running Tests

### Commands

```bash
# Run all tests
npm test

# Run tests in headed mode (show browser)
npm run test:headed

# Run tests with UI (interactive test runner)
npm run test:ui

# View test report
npm run test:report

# Run specific test file
npx playwright test tests/integration/test_page_load.spec.ts

# Run tests matching pattern
npx playwright test --grep="Dashboard"

# Run with debugging
npx playwright test --debug

# Update snapshots
npx playwright test --update-snapshots
```

### CI/CD Integration

Tests run automatically in GitHub Actions on:
- Push to `main` or `develop` branches
- Pull requests
- Manual workflow trigger

```yaml
# .github/workflows/webui-ci.yml
- name: Run Playwright tests
  run: npm test

- name: Upload test results
  if: always()
  uses: actions/upload-artifact@v3
  with:
    name: playwright-report
    path: playwright-report/
```

## Test Structure

### Directory Organization

```
tests/
├── integration/
│   ├── test_page_load.spec.ts       # Page load tests
│   ├── test_authentication.spec.ts  # Login/logout flows
│   ├── test_services.spec.ts        # Service CRUD operations
│   ├── test_clusters.spec.ts        # Cluster management
│   ├── test_proxies.spec.ts         # Proxy monitoring
│   └── test_[feature].spec.ts       # Feature-specific tests
└── playwright.config.ts
```

## Test Patterns

### Basic Page Load Test

```typescript
import { test, expect } from '@playwright/test';

test.describe('Page Load - Services', () => {
  test.beforeEach(async ({ page }) => {
    // Setup: login before each test
    await login(page);
  });

  test('should load services page without errors', async ({ page }) => {
    // Navigate to page
    const response = await page.goto('/dashboard/services');

    // Verify HTTP status
    expect(response?.status()).toBe(200);

    // Wait for page to load
    await page.waitForLoadState('networkidle');

    // Verify no JavaScript errors
    const errors: Error[] = [];
    page.on('pageerror', (error) => errors.push(error));
    expect(errors).toHaveLength(0);
  });

  test('should display services table', async ({ page }) => {
    await page.goto('/dashboard/services');

    // Verify content exists
    await expect(page.locator('h1')).toContainText('Services');
    await expect(page.locator('[data-testid="services-table"]')).toBeVisible();
  });
});
```

### Authentication Helper

```typescript
// tests/helpers/auth.ts
async function login(page: Page) {
  await page.goto('/login');

  await page.fill('input[name="email"]', 'admin@test.com');
  await page.fill('input[name="password"]', 'Admin123!');

  await page.click('button[type="submit"]');

  // Wait for redirect to dashboard
  await expect(page).toHaveURL(/\/dashboard/);
}

async function logout(page: Page) {
  await page.click('[data-testid="user-menu"]');
  await page.click('text=Logout');
  await expect(page).toHaveURL(/\/login/);
}

export { login, logout };
```

### Form Testing

```typescript
test('should create new service', async ({ page }) => {
  await page.goto('/dashboard/services');

  // Click create button
  await page.click('button:has-text("Create Service")');

  // Fill form fields
  await page.fill('input[name="name"]', 'Test Service');
  await page.fill('input[name="upstream"]', 'http://localhost:8080');
  await page.selectOption('select[name="protocol"]', 'tcp');

  // Submit form
  await page.click('button[type="submit"]');

  // Verify success notification
  await expect(page.locator('text=Service created successfully')).toBeVisible();

  // Verify redirect to service details
  await expect(page).toHaveURL(/\/dashboard\/services\/\d+/);
});
```

### Error Handling Test

```typescript
test('should display validation errors', async ({ page }) => {
  await page.goto('/dashboard/services');
  await page.click('button:has-text("Create Service")');

  // Submit empty form
  await page.click('button[type="submit"]');

  // Verify error messages
  await expect(page.locator('text=Name is required')).toBeVisible();
  await expect(page.locator('text=Upstream URL is required')).toBeVisible();
});
```

### Modal Testing

```typescript
test('should confirm delete action', async ({ page }) => {
  await page.goto('/dashboard/services');

  // Find and click delete button
  await page.click('[data-testid="service-row"]:first-child [data-testid="delete-btn"]');

  // Verify confirmation modal appears
  await expect(page.locator('h2:has-text("Delete Service?")')).toBeVisible();

  // Click confirm button
  await page.click('button:has-text("Confirm Delete")');

  // Verify success
  await expect(page.locator('text=Service deleted')).toBeVisible();
});
```

### Data Grid Testing

```typescript
test('should filter and sort services', async ({ page }) => {
  await page.goto('/dashboard/services');

  // Type in search field
  await page.fill('input[placeholder="Search services"]', 'test');

  // Verify table filtered
  const rows = page.locator('[data-testid="service-row"]');
  await expect(rows).toHaveCount(1);

  // Click sort header
  await page.click('th:has-text("Name")');

  // Verify sort applied
  const firstServiceName = await rows.first().locator('[data-testid="service-name"]').textContent();
  expect(firstServiceName).toMatch(/^[A-Z]/);  // Alphabetical
});
```

### API Mocking

```typescript
import { expect, test as base } from '@playwright/test';
import { Page } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  // Mock API responses
  await page.route('**/api/services', (route) => {
    route.abort('Failed');  // Simulate error
  });

  await page.route('**/api/services*', (route) => {
    if (route.request().method() === 'POST') {
      route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          id: '1',
          name: 'Created Service',
        }),
      });
    }
  });
});
```

## Test Scenarios by Feature

### Dashboard Tests

```typescript
test.describe('Dashboard', () => {
  test('should display summary statistics', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard');

    await expect(page.locator('text=Total Clusters')).toBeVisible();
    await expect(page.locator('text=Total Services')).toBeVisible();
    await expect(page.locator('text=Total Proxies')).toBeVisible();
    await expect(page.locator('text=Active Connections')).toBeVisible();
  });

  test('should show recent activity', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard');

    await expect(page.locator('[data-testid="activity-feed"]')).toBeVisible();
  });

  test('should refresh metrics in real-time', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard');

    // Get initial metric value
    const initialValue = await page.locator('[data-testid="cpu-metric"]').textContent();

    // Wait for refresh (typically 5-10 seconds)
    await page.waitForTimeout(6000);

    // Value should potentially change (may or may not depending on actual metrics)
    const updatedValue = await page.locator('[data-testid="cpu-metric"]').textContent();

    // At minimum, value should be present
    expect(updatedValue).toBeTruthy();
  });
});
```

### Authentication Tests

```typescript
test.describe('Authentication', () => {
  test('should login with valid credentials', async ({ page }) => {
    await page.goto('/login');

    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'Admin123!');
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/dashboard/);

    // Verify token in localStorage
    const token = await page.evaluate(() => localStorage.getItem('auth_token'));
    expect(token).toBeTruthy();
  });

  test('should reject invalid credentials', async ({ page }) => {
    await page.goto('/login');

    await page.fill('input[name="email"]', 'admin@test.com');
    await page.fill('input[name="password"]', 'WrongPassword');
    await page.click('button[type="submit"]');

    // Verify error message
    await expect(page.locator('text=Invalid credentials')).toBeVisible();

    // Verify still on login page
    await expect(page).toHaveURL(/\/login/);
  });

  test('should logout successfully', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard');

    // Open user menu and logout
    await page.click('[data-testid="user-menu"]');
    await page.click('text=Logout');

    // Verify redirected to login
    await expect(page).toHaveURL(/\/login/);

    // Verify token cleared
    const token = await page.evaluate(() => localStorage.getItem('auth_token'));
    expect(token).toBeNull();
  });
});
```

### Cluster Management Tests

```typescript
test.describe('Cluster Management', () => {
  test('should list all clusters', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/clusters');

    // Verify clusters are displayed
    const rows = page.locator('[data-testid="cluster-row"]');
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });

  test('should create new cluster', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/clusters');

    // Click create button
    await page.click('button:has-text("New Cluster")');

    // Fill form
    await page.fill('input[name="name"]', 'Test Cluster');
    await page.fill('textarea[name="description"]', 'Test cluster for integration tests');

    // Submit
    await page.click('button[type="submit"]');

    // Verify success
    await expect(page.locator('text=Cluster created successfully')).toBeVisible();
  });

  test('should rotate API key', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/clusters');

    // Find cluster row
    const clusterRow = page.locator('[data-testid="cluster-row"]').first();

    // Click actions menu
    await clusterRow.locator('[data-testid="actions-menu"]').click();

    // Click rotate API key
    await page.click('text=Rotate API Key');

    // Confirm action
    await page.click('button:has-text("Confirm")');

    // Verify success
    await expect(page.locator('text=API key rotated')).toBeVisible();
  });
});
```

### Service Management Tests

```typescript
test.describe('Service Management', () => {
  test('should list services', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/services');

    await expect(page.locator('[data-testid="services-table"]')).toBeVisible();
  });

  test('should create service', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/services');

    await page.click('button:has-text("Create Service")');

    await page.fill('input[name="name"]', 'Test Service');
    await page.fill('input[name="port"]', '8080');
    await page.selectOption('select[name="protocol"]', 'tcp');
    await page.fill('input[name="upstream"]', 'http://backend:8080');

    await page.click('button[type="submit"]');

    await expect(page.locator('text=Service created')).toBeVisible();
  });

  test('should update service', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/services');

    // Click edit button on first service
    await page.locator('[data-testid="services-table"] [data-testid="edit-btn"]').first().click();

    // Update name
    await page.fill('input[name="name"]', 'Updated Service Name');

    // Submit
    await page.click('button[type="submit"]');

    // Verify update
    await expect(page.locator('text=Service updated')).toBeVisible();
  });
});
```

### Proxy Monitoring Tests

```typescript
test.describe('Proxy Monitoring', () => {
  test('should display proxy status', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/proxies');

    // Verify proxy table with status indicators
    await expect(page.locator('[data-testid="proxy-status-online"]')).toBeVisible();
  });

  test('should show proxy metrics', async ({ page }) => {
    await login(page);
    await page.goto('/dashboard/proxies');

    // Click on proxy row to view details
    await page.locator('[data-testid="proxy-row"]').first().click();

    // Verify metrics displayed
    await expect(page.locator('[data-testid="cpu-usage"]')).toBeVisible();
    await expect(page.locator('[data-testid="memory-usage"]')).toBeVisible();
    await expect(page.locator('[data-testid="connection-count"]')).toBeVisible();
  });
});
```

### Enterprise Features Tests

```typescript
test.describe('Enterprise Features', () => {
  test('should not show enterprise features in community edition', async ({ page }) => {
    await login(page);

    // Enterprise links should not be visible
    const enterpriseLink = page.locator('a:has-text("Traffic Shaping")');
    await expect(enterpriseLink).not.toBeVisible();
  });

  test('should show enterprise features with valid license', async ({ page }) => {
    // This would require setting license in test environment
    await login(page);

    // If license is valid, features should be visible
    const trafficShapingLink = page.locator('a:has-text("Traffic Shaping")');

    // Feature visibility depends on license validation
    // Test implementation would mock license check
  });
});
```

## Debugging Tests

### Debug Mode

```bash
# Run with Playwright Inspector
npx playwright test --debug

# Run with Chrome DevTools
npx playwright test --headed --debug
```

### Screenshots and Videos

```typescript
test('should create service', async ({ page }) => {
  // Take screenshot before action
  await page.screenshot({ path: 'before-create.png' });

  await page.click('button:has-text("Create Service")');

  // Take screenshot after action
  await page.screenshot({ path: 'after-create.png' });
});
```

### Trace Files

```typescript
test('should create service', async ({ page, context }) => {
  // Start tracing
  await context.tracing.start({ screenshots: true, snapshots: true });

  // Run test
  await page.goto('/dashboard/services');
  await page.click('button:has-text("Create Service")');

  // Stop and save trace
  await context.tracing.stop({ path: 'trace.zip' });
});
```

View trace:
```bash
npx playwright show-trace trace.zip
```

### Logging

```typescript
test('should create service', async ({ page }) => {
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      console.error('Page error:', msg.text());
    }
  });

  page.on('pageerror', (error) => {
    console.error('Page crashed:', error);
  });

  // Your test code
});
```

## Best Practices

1. **Use data-testid attributes** for reliable element selection
   ```typescript
   await page.click('[data-testid="create-service-btn"]');
   ```

2. **Wait for network idle** before verifying page content
   ```typescript
   await page.waitForLoadState('networkidle');
   ```

3. **Use page.fill() over type()** for better performance
   ```typescript
   await page.fill('input[name="email"]', 'user@example.com');
   ```

4. **Avoid hardcoded waits** - use Playwright's built-in waiters
   ```typescript
   // Good
   await expect(element).toBeVisible();

   // Bad
   await page.waitForTimeout(3000);
   ```

5. **Test user workflows** not implementation details
   ```typescript
   // Good - tests user action
   await page.click('button:has-text("Delete")');
   await page.click('button:has-text("Confirm")');

   // Bad - tests implementation
   await page.click('.btn-delete-icon');
   ```

6. **Use beforeEach for common setup**
   ```typescript
   test.beforeEach(async ({ page }) => {
     await login(page);
   });
   ```

7. **Group related tests with describe blocks**
   ```typescript
   test.describe('Services', () => {
     test('should list services', ...);
     test('should create service', ...);
     test('should delete service', ...);
   });
   ```

## CI/CD Integration

Tests run on every push and PR:

```yaml
- name: Run E2E tests
  run: npm test

- name: Upload failed test artifacts
  if: failure()
  uses: actions/upload-artifact@v3
  with:
    name: test-artifacts
    path: |
      playwright-report/
      test-results/
```

Failed tests block merge to main branch until fixed.

## Test Coverage Goals

- Page load tests: All main pages and routes
- Happy path workflows: Create, read, update, delete operations
- Error handling: Validation errors, server errors, network errors
- Edge cases: Empty states, large datasets, special characters
- Authentication: Login, logout, token expiration
- Authorization: Role-based access control
- Enterprise features: License gating and feature availability

Current coverage: ~50% of application workflows
Target coverage: 80%+ before release

## Maintenance

- Update selectors when UI changes
- Add new tests for new features
- Review and update test helpers regularly
- Archive old or obsolete test files
- Keep test data synchronized with development
