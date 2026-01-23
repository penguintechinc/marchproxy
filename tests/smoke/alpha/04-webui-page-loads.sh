#!/bin/bash
# Alpha Smoke Test 4: WebUI page and tab loads
# Verifies all pages load without JavaScript errors

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo "=========================================="
echo "Alpha Smoke Test 4: WebUI Page Loads"
echo "=========================================="
echo ""

# Check if node is available for Playwright/Puppeteer
if ! command -v node &> /dev/null; then
    echo "⚠️  Node.js not found - skipping WebUI page load tests"
    echo "   Install Node.js to run full WebUI smoke tests"
    exit 0
fi

# Check if WebUI is running
if ! curl -f -s http://localhost:3000 > /dev/null 2>&1; then
    echo "⚠️  WebUI not responding at http://localhost:3000"
    echo "   Start WebUI with 'cd webui && npm run dev' to test"
    exit 0
fi

FAILED=0

echo "Checking if Playwright is installed..."
cd "$PROJECT_ROOT/webui"

if [ ! -d "node_modules/playwright" ]; then
    echo "⚠️  Playwright not installed - installing now..."
    npm install -D @playwright/test
    npx playwright install chromium --with-deps
fi

# Create temporary test script
cat > /tmp/webui-smoke-test.js << 'EOF'
const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  let failed = 0;

  // Track console errors
  const errors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });

  page.on('pageerror', err => {
    errors.push(err.toString());
  });

  // Test pages
  const pages = [
    { url: 'http://localhost:3000/', name: 'Home/Login' },
    { url: 'http://localhost:3000/dashboard', name: 'Dashboard' },
    { url: 'http://localhost:3000/clusters', name: 'Clusters' },
    { url: 'http://localhost:3000/proxies', name: 'Proxies' },
    { url: 'http://localhost:3000/services', name: 'Services' },
    { url: 'http://localhost:3000/users', name: 'Users' },
    { url: 'http://localhost:3000/roles', name: 'Roles (RBAC)' },
  ];

  for (const pageInfo of pages) {
    try {
      console.log(`Checking ${pageInfo.name} page...`);
      errors.length = 0; // Reset errors

      await page.goto(pageInfo.url, { waitUntil: 'networkidle', timeout: 10000 });

      // Wait a bit for any async errors
      await page.waitForTimeout(1000);

      if (errors.length > 0) {
        console.log(`❌ ${pageInfo.name} has JavaScript errors:`);
        errors.forEach(err => console.log(`   ${err}`));
        failed++;
      } else {
        console.log(`✅ ${pageInfo.name} loaded successfully`);
      }
    } catch (err) {
      console.log(`❌ ${pageInfo.name} failed to load: ${err.message}`);
      failed++;
    }
  }

  await browser.close();

  console.log('');
  if (failed === 0) {
    console.log('✅ All pages loaded successfully');
    process.exit(0);
  } else {
    console.log(`❌ ${failed} page(s) failed to load`);
    process.exit(1);
  }
})();
EOF

# Run the test
echo ""
node /tmp/webui-smoke-test.js
EXIT_CODE=$?

# Cleanup
rm /tmp/webui-smoke-test.js

exit $EXIT_CODE
