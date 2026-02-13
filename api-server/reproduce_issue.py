
import sys
import os
from passlib.context import CryptContext

# 1. Setup the fallback context (simulating the migration failure case)
fallback_context = CryptContext(schemes=["bcrypt"], deprecated="auto")
def fallback_hash(password: str) -> str:
    return fallback_context.hash(password)

# 2. Setup the app's context (simulating the compiled app)
# We try to import it, but if it fails we mock it to what we see in the file
try:
    sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
    from app.core.security import verify_password as app_verify
    from app.core.security import get_password_hash as app_hash
    print("Successfully imported app.core.security")
except ImportError as e:
    print(f"Failed to import app.core.security: {e}")
    print("Using mocked app context based on file reading")
    app_context = CryptContext(schemes=["bcrypt"], deprecated="auto")
    def app_verify(plain, hashed):
        return app_context.verify(plain, hashed)
    def app_hash(password):
        return app_context.hash(password)

# 3. Test
password = "admin123"
print(f"Testing password: {password}")

# Generate hash using fallback
h_fallback = fallback_hash(password)
print(f"Fallback Hash: {h_fallback}")

# Verify using app logic
is_valid = app_verify(password, h_fallback)
print(f"App verify(Fallback Hash) -> {is_valid}")

if not is_valid:
    print("FAIL: Fallback hash is NOT valid under App verification.")
else:
    print("SUCCESS: Fallback hash IS valid under App verification.")

# Generate hash using app logic
h_app = app_hash(password)
print(f"App Hash: {h_app}")
is_valid_app = app_verify(password, h_app)
print(f"App verify(App Hash) -> {is_valid_app}")
