#!/bin/bash
# Setup script for MarchProxy website submodule with sparse-checkout
# Only includes marchproxy/ and marchproxy-docs/ directories

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_status "Setting up MarchProxy website submodule with sparse-checkout..."

# Check if we're in the right directory
if [ ! -f ".gitmodules" ]; then
    echo "❌ Error: .gitmodules file not found. Please run this script from the MarchProxy root directory."
    exit 1
fi

# Check if website submodule already exists
if [ -d "website" ]; then
    print_warning "Website directory already exists. Configuring sparse-checkout..."

    cd website

    # Enable sparse checkout
    git config core.sparseCheckout true

    # Create sparse-checkout file
    echo "marchproxy/" > .git/info/sparse-checkout
    echo "marchproxy-docs/" >> .git/info/sparse-checkout

    # Apply sparse checkout
    git read-tree -m -u HEAD

    cd ..

    print_success "Sparse-checkout configured for existing website submodule"
else
    print_status "Adding website submodule with sparse-checkout..."

    # Add submodule
    git submodule add https://github.com/penguintechinc/website.git website

    cd website

    # Enable sparse checkout
    git config core.sparseCheckout true

    # Create sparse-checkout file
    echo "marchproxy/" > .git/info/sparse-checkout
    echo "marchproxy-docs/" >> .git/info/sparse-checkout

    # Apply sparse checkout
    git read-tree -m -u HEAD

    cd ..

    print_success "Website submodule added with sparse-checkout"
fi

# Verify sparse checkout is working
if [ -d "website/marchproxy" ] && [ -d "website/marchproxy-docs" ]; then
    print_success "Sparse-checkout verified: marchproxy/ and marchproxy-docs/ directories present"
else
    echo "❌ Error: Sparse-checkout may not be working correctly"
    exit 1
fi

# Check what directories are present
print_status "Website submodule contents:"
ls -la website/ | while read line; do
    echo "  $line"
done

print_success "Website submodule setup complete!"
print_status ""
print_status "Next steps:"
print_status "1. Update marchproxy/ website with dual proxy features"
print_status "2. Update marchproxy-docs/ with new documentation"
print_status "3. See WEBSITE_UPDATE_CHECKLIST.md for detailed changes needed"
print_status ""
print_status "Working with the submodule:"
print_status "  cd website/marchproxy      # Work on main website"
print_status "  cd website/marchproxy-docs # Work on documentation"