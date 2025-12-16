# AILB Testing Guide

## Quick Start

```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
python3 -m pytest tests/ -v -o addopts=""
```

## Run Specific Test Suites

### Run only Prompt Security Scanner tests
```bash
python3 -m pytest tests/test_prompt_security.py -v -o addopts=""
```

### Run only Token Manager tests
```bash
python3 -m pytest tests/test_token_manager.py -v -o addopts=""
```

## Run Specific Test Classes

### Prompt Security Tests

```bash
# Test scanner initialization
python3 -m pytest tests/test_prompt_security.py::TestPromptSecurityScannerInitialization -v -o addopts=""

# Test injection detection
python3 -m pytest tests/test_prompt_security.py::TestPromptInjectionDetection -v -o addopts=""

# Test jailbreak detection
python3 -m pytest tests/test_prompt_security.py::TestJailbreakDetection -v -o addopts=""

# Test data extraction detection
python3 -m pytest tests/test_prompt_security.py::TestDataExtractionDetection -v -o addopts=""

# Test prompt length validation
python3 -m pytest tests/test_prompt_security.py::TestPromptLengthValidation -v -o addopts=""

# Test blocking decisions
python3 -m pytest tests/test_prompt_security.py::TestShouldBlockDecision -v -o addopts=""

# Test sanitization
python3 -m pytest tests/test_prompt_security.py::TestSanitizePrompt -v -o addopts=""

# Test security policies
python3 -m pytest tests/test_prompt_security.py::TestSecurityPolicies -v -o addopts=""

# Test custom patterns
python3 -m pytest tests/test_prompt_security.py::TestCustomPatterns -v -o addopts=""

# Test rate limiting
python3 -m pytest tests/test_prompt_security.py::TestRateLimiting -v -o addopts=""

# Test statistics
python3 -m pytest tests/test_prompt_security.py::TestStatistics -v -o addopts=""

# Test message scanning
python3 -m pytest tests/test_prompt_security.py::TestMessageScanning -v -o addopts=""

# Test security logging
python3 -m pytest tests/test_prompt_security.py::TestSecurityLogging -v -o addopts=""

# Test credential harvesting detection
python3 -m pytest tests/test_prompt_security.py::TestCredentialHarvesting -v -o addopts=""
```

### Token Manager Tests

```bash
# Test initialization
python3 -m pytest tests/test_token_manager.py::TestTokenManagerInitialization -v -o addopts=""

# Test token counting
python3 -m pytest tests/test_token_manager.py::TestTokenCounting -v -o addopts=""

# Test token conversion
python3 -m pytest tests/test_token_manager.py::TestTokenConversion -v -o addopts=""

# Test cost calculation
python3 -m pytest tests/test_token_manager.py::TestCostCalculation -v -o addopts=""

# Test usage processing
python3 -m pytest tests/test_token_manager.py::TestUsageProcessing -v -o addopts=""

# Test quota configuration
python3 -m pytest tests/test_token_manager.py::TestQuotaConfiguration -v -o addopts=""

# Test quota enforcement
python3 -m pytest tests/test_token_manager.py::TestQuotaEnforcement -v -o addopts=""

# Test usage statistics
python3 -m pytest tests/test_token_manager.py::TestUsageStatistics -v -o addopts=""

# Test conversion rates
python3 -m pytest tests/test_token_manager.py::TestConversionRates -v -o addopts=""

# Test thread safety
python3 -m pytest tests/test_token_manager.py::TestThreadSafety -v -o addopts=""

# Test edge cases
python3 -m pytest tests/test_token_manager.py::TestEdgeCases -v -o addopts=""
```

## Run Individual Tests

```bash
# Run specific test method
python3 -m pytest tests/test_prompt_security.py::TestPromptSecurityScannerInitialization::test_create_scanner_with_strict_policy -v -o addopts=""

# Run multiple tests matching a pattern
python3 -m pytest tests/ -k "detection" -v -o addopts=""
```

## Test Output Options

### Verbose output with full details
```bash
python3 -m pytest tests/ -vv -o addopts=""
```

### Show only failed tests
```bash
python3 -m pytest tests/ --tb=short -o addopts=""
```

### Show print statements during tests
```bash
python3 -m pytest tests/ -s -o addopts=""
```

### Stop after first failure
```bash
python3 -m pytest tests/ -x -o addopts=""
```

### Show slowest 10 tests
```bash
python3 -m pytest tests/ --durations=10 -o addopts=""
```

## Understanding Test Results

### PASSED ✓
Test executed successfully and all assertions passed.

### FAILED ✗
Test executed but one or more assertions failed. Check the error message for details.

### Known Failures

See `TEST_REPORT.md` for details on currently failing tests:

#### Prompt Security Scanner (15 failures)
- Pattern detection thresholds in balanced/permissive policies too high
- Sanitization not being applied as expected
- Statistics logging depends on threat detection

#### Token Manager (3 failures)
- Quota enforcement not blocking when limits exceeded
- Token counting may be underestimating usage

## Test Structure

### Test Organization
- `/home/penguin/code/MarchProxy/proxy-ailb/tests/__init__.py` - Package marker
- `/home/penguin/code/MarchProxy/proxy-ailb/tests/test_prompt_security.py` - Security scanning tests (820 lines, 74 tests)
- `/home/penguin/code/MarchProxy/proxy-ailb/tests/test_token_manager.py` - Token management tests (979 lines, 63 tests)

### Test Naming Convention
- **Test files**: `test_*.py`
- **Test classes**: `Test*` (e.g., `TestPromptSecurityScanner`)
- **Test methods**: `test_*` (descriptive names)

### Test Dependencies
- pytest 9.0.2
- Python 3.12.3
- No external dependencies required

## Adding New Tests

### Template for new test class
```python
class TestNewFeature:
    """Test description"""

    def test_basic_functionality(self):
        """Test basic behavior"""
        # Arrange
        component = ComponentClass()

        # Act
        result = component.method()

        # Assert
        assert result is expected_value
```

### Best Practices
1. Use descriptive test names
2. Test one concept per test method
3. Include docstrings explaining what is tested
4. Use arrange-act-assert pattern
5. Test both success and failure cases
6. Include edge case tests

## Debugging Failed Tests

### Get detailed failure information
```bash
python3 -m pytest tests/test_prompt_security.py::TestSomeClass::test_some_method -vv -o addopts=""
```

### Run with Python debugger
```bash
python3 -m pytest tests/ --pdb -o addopts=""
```

### See captured output
```bash
python3 -m pytest tests/ -s -o addopts=""
```

## Performance

Typical test execution time:
- Full test suite: ~1.5 seconds
- Prompt Security tests only: ~0.8 seconds
- Token Manager tests only: ~0.7 seconds

## CI/CD Integration

Tests can be integrated into CI/CD pipelines:

```bash
# Run tests with exit code for CI
python3 -m pytest tests/ -o addopts="" --tb=short

# Generate coverage reports
python3 -m pytest tests/ -o addopts="" --cov=app --cov-report=html
```

## Troubleshooting

### pytest: command not found
```bash
python3 -m pip install pytest
```

### Import errors
Ensure you're running from the correct directory:
```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
```

### Configuration errors
The pyproject.toml at the parent level has coverage settings. Use `-o addopts=""` to ignore.

## Resources

- **Pytest Documentation**: https://docs.pytest.org/
- **Test Report**: See `TEST_REPORT.md` for detailed results and failure analysis

---

Last Updated: 2025-12-15
