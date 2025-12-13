import React from 'react';
import {
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Box,
  Divider,
} from '@mui/material';
import {
  Dashboard as DashboardIcon,
  Storage as ClusterIcon,
  Dns as ServiceIcon,
  Router as ProxyIcon,
  Settings as SettingsIcon,
  BarChart as MetricsIcon,
  Security as CertificateIcon,
  Speed as TracingIcon,
  NotificationsActive as AlertsIcon,
} from '@mui/icons-material';
import { useNavigate, useLocation } from 'react-router-dom';

interface SidebarProps {
  drawerWidth: number;
  mobileOpen: boolean;
  onClose: () => void;
  isMobile: boolean;
}

interface MenuItem {
  text: string;
  icon: React.ReactElement;
  path: string;
}

const menuItems: MenuItem[] = [
  { text: 'Dashboard', icon: <DashboardIcon />, path: '/dashboard' },
  { text: 'Clusters', icon: <ClusterIcon />, path: '/clusters' },
  { text: 'Services', icon: <ServiceIcon />, path: '/services' },
  { text: 'Proxies', icon: <ProxyIcon />, path: '/proxies' },
  { text: 'Certificates', icon: <CertificateIcon />, path: '/certificates' },
  { text: 'Tracing', icon: <TracingIcon />, path: '/observability/tracing' },
  { text: 'Metrics', icon: <MetricsIcon />, path: '/observability/metrics' },
  { text: 'Alerts', icon: <AlertsIcon />, path: '/observability/alerts' },
  { text: 'Settings', icon: <SettingsIcon />, path: '/settings' },
];

const Sidebar: React.FC<SidebarProps> = ({
  drawerWidth,
  mobileOpen,
  onClose,
  isMobile,
}) => {
  const navigate = useNavigate();
  const location = useLocation();

  const handleNavigation = (path: string) => {
    navigate(path);
    if (isMobile) {
      onClose();
    }
  };

  const drawer = (
    <Box>
      <Toolbar>
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '100%',
          }}
        >
          <Box
            component="img"
            src="/marchproxy-logo.png"
            alt="MarchProxy"
            sx={{ height: 40, objectFit: 'contain' }}
            onError={(e: any) => {
              e.target.style.display = 'none';
            }}
          />
        </Box>
      </Toolbar>
      <Divider />
      <List>
        {menuItems.map((item) => (
          <ListItem key={item.text} disablePadding>
            <ListItemButton
              selected={location.pathname === item.path}
              onClick={() => handleNavigation(item.path)}
              sx={{
                '&.Mui-selected': {
                  bgcolor: 'primary.dark',
                  '&:hover': {
                    bgcolor: 'primary.dark',
                  },
                },
              }}
            >
              <ListItemIcon sx={{ color: 'inherit' }}>{item.icon}</ListItemIcon>
              <ListItemText primary={item.text} />
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Box>
  );

  return (
    <Box
      component="nav"
      sx={{ width: { sm: drawerWidth }, flexShrink: { sm: 0 } }}
    >
      {isMobile ? (
        <Drawer
          variant="temporary"
          open={mobileOpen}
          onClose={onClose}
          ModalProps={{
            keepMounted: true,
          }}
          sx={{
            display: { xs: 'block', sm: 'none' },
            '& .MuiDrawer-paper': {
              boxSizing: 'border-box',
              width: drawerWidth,
            },
          }}
        >
          {drawer}
        </Drawer>
      ) : (
        <Drawer
          variant="permanent"
          sx={{
            display: { xs: 'none', sm: 'block' },
            '& .MuiDrawer-paper': {
              boxSizing: 'border-box',
              width: drawerWidth,
            },
          }}
          open
        >
          {drawer}
        </Drawer>
      )}
    </Box>
  );
};

export default Sidebar;
