import React from 'react';
import { useNavigate } from 'react-router-dom';
import { LoginPageBuilder, type LoginPageBuilderProps } from '@penguintechinc/react-libs';
import { useAuthStore } from '@store/authStore';

const Login: React.FC = () => {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuthStore();

  React.useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard');
    }
  }, [isAuthenticated, navigate]);

  const handleLoginSuccess: LoginPageBuilderProps['onSuccess'] = (response) => {
    // LoginPageBuilder handles token storage via the API response
    // After successful login, redirect to dashboard
    console.log('[Login] Authentication successful');
    navigate('/dashboard');
  };

  const handleLoginError = (error: unknown) => {
    const message = error instanceof Error ? error.message : 'Login failed';
    console.error('[Login] Authentication error', message);
  };

  return (
    <LoginPageBuilder
      api={{
        loginEndpoint: '/api/v1/auth/login',
        method: 'post',
      }}
      branding={{
        appName: 'MarchProxy',
        logoUrl: '/marchproxy-logo.png',
        logoHeight: 300,
        footerText: 'Enterprise Proxy and Load Balancer Suite',
      }}
      onSuccess={handleLoginSuccess}
      onError={handleLoginError}
      mfa={{
        enabled: true,
        label: '2FA Code',
        helperText: 'Enter your 6-digit TOTP code',
      }}
      themeMode="dark"
      showRememberMe={true}
      showForgotPassword={false}
      showSignUp={false}
    />
  );
};

export default Login;
