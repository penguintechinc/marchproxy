# AILB Testing Guide

## Quick Start

```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
python3 -m pytest tests/ -v
```

## Test Structure

```
tests/
├── test_prompt_security.py    # 820 lines, 74 tests
├── test_token_manager.py      # 979 lines, 63 tests
└── __init__.py
```

**Coverage Requirements:**
- Minimum 80% code coverage
- All critical paths must have tests
- Security features must be thoroughly tested

## Running Tests

### All Tests
```bash
python3 -m pytest tests/ -v
```

### Specific Test File
```bash
python3 -m pytest tests/test_prompt_security.py -v
python3 -m pytest tests/test_token_manager.py -v
```

### Specific Test Class
```bash
python3 -m pytest tests/test_prompt_security.py::TestPromptSecurityScannerInitialization -v
python3 -m pytest tests/test_token_manager.py::TestTokenCounting -v
```

### Specific Test Method
```bash
python3 -m pytest tests/test_prompt_security.py::TestPromptInjectionDetection::test_detect_prompt_injection -v
```

### Pattern Matching
```bash
python3 -m pytest tests/ -k "detection" -v
python3 -m pytest tests/ -k "quota" -v
```

## Test Output Options

### Verbose Output
```bash
python3 -m pytest tests/ -vv
```

### Show Print Statements
```bash
python3 -m pytest tests/ -s
```

### Stop on First Failure
```bash
python3 -m pytest tests/ -x
```

### Show Slowest Tests
```bash
python3 -m pytest tests/ --durations=10
```

### Generate Coverage Report
```bash
python3 -m pytest tests/ --cov=app --cov-report=html
```

### Exit Code Only (CI)
```bash
python3 -m pytest tests/ --tb=short
```

## Test Categories

### Prompt Security Tests (test_prompt_security.py)

Tests for prompt injection detection, jailbreak prevention, and sanitization.

**Test Classes:**
- `TestPromptSecurityScannerInitialization` - Scanner setup
- `TestPromptInjectionDetection` - Injection attack detection
- `TestJailbreakDetection` - Jailbreak attempt detection
- `TestDataExtractionDetection` - Credential/data extraction detection
- `TestPromptLengthValidation` - Length validation
- `TestShouldBlockDecision` - Block/allow decision logic
- `TestSanitizePrompt` - Prompt sanitization
- `TestSecurityPolicies` - Policy application (strict/balanced/permissive)
- `TestCustomPatterns` - Custom pattern detection
- `TestRateLimiting` - Rate limit enforcement
- `TestStatistics` - Threat statistics tracking
- `TestMessageScanning` - Multi-message scanning
- `TestSecurityLogging` - Security event logging
- `TestCredentialHarvesting` - Credential harvesting detection

### Token Manager Tests (test_token_manager.py)

Tests for token counting, cost calculation, and quota enforcement.

**Test Classes:**
- `TestTokenManagerInitialization` - Manager setup
- `TestTokenCounting` - Token counting accuracy
- `TestTokenConversion` - Token conversion between providers
- `TestCostCalculation` - Cost calculation
- `TestUsageProcessing` - Usage record processing
- `TestQuotaConfiguration` - Quota setup
- `TestQuotaEnforcement` - Quota limit enforcement
- `TestUsageStatistics` - Usage statistics
- `TestConversionRates` - Provider conversion rates
- `TestThreadSafety` - Concurrent access
- `TestEdgeCases` - Boundary conditions

## Adding New Tests

### Template
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
2. Test one concept per method
3. Include docstrings
4. Use arrange-act-assert pattern
5. Test success and failure cases
6. Include edge cases

### Example Test
```python
class TestBudgetEnforcement:
    """Test budget limit enforcement"""

    def test_reject_request_exceeding_budget(self):
        """Verify requests over budget are rejected"""
        # Arrange
        tracker = CostTracker()
        tracker.set_budget("key-123", 100.0)

        # Act
        can_proceed = tracker.check_budget("key-123", 150.0)

        # Assert
        assert not can_proceed
```

## Known Test Issues

See `TEST_REPORT.md` for details on currently failing tests.

**Prompt Security Scanner Issues:**
- 15 test failures
- Pattern thresholds too high for balanced/permissive policies
- Sanitization not applied as expected
- Statistics depend on threat detection

**Token Manager Issues:**
- 3 test failures
- Quota enforcement not blocking at limits
- Token counting may underestimate usage

## Debugging Failed Tests

### Detailed Output
```bash
python3 -m pytest tests/test_file.py::TestClass::test_method -vv
```

### With Python Debugger
```bash
python3 -m pytest tests/ --pdb
```

### Capture Output
```bash
python3 -m pytest tests/ -s
```

### Show Only Failed Tests
```bash
python3 -m pytest tests/ --tb=short
```

## Test Performance

Expected execution times:
- Full suite: ~1.5 seconds
- Prompt Security: ~0.8 seconds
- Token Manager: ~0.7 seconds

## CI/CD Integration

### Basic Test Run
```bash
python3 -m pytest tests/ --tb=short
```

### With Coverage Report
```bash
python3 -m pytest tests/ --cov=app --cov-report=term-missing
```

### Fail on Coverage
```bash
python3 -m pytest tests/ --cov=app --cov-fail-under=80
```

## Troubleshooting

### Import Errors
Ensure you're in the correct directory:
```bash
cd /home/penguin/code/MarchProxy/proxy-ailb
```

### pytest Not Found
Install pytest:
```bash
pip install pytest
```

### Hanging Tests
Add timeout:
```bash
pip install pytest-timeout
python3 -m pytest tests/ --timeout=10
```

## Test Dependencies

- Python 3.12.3
- pytest 9.0.2
- No external API dependencies (mocked)

## Resources

- **Pytest Documentation**: https://docs.pytest.org/
- **Test Report**: See `TEST_REPORT.md`
- **Architecture**: See `ARCHITECTURE.md`

---

**Last Updated:** 2025-12-15
