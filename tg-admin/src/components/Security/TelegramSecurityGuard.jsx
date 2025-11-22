import React, { useState, useEffect } from 'react';
import { Container, Paper, Text, Stack, Alert, Loader, Button, Group, Badge, } from '@mantine/core';
import { IconShield, IconShieldX, IconAlertTriangle, IconRefresh } from '@tabler/icons-react';
import { useTelegramSecurity, checkAdminStatus } from '../../hooks/useTelegramSecurity';
import { shouldUseDevelopmentMode, devLog } from '../../lib/dev-utils';
const TelegramSecurityGuard = ({ children }) => {
    const { isValidEnvironment, isLoading, error, telegramId } = useTelegramSecurity();
    const [isAdmin, setIsAdmin] = useState(null);
    const [adminCheckLoading, setAdminCheckLoading] = useState(false);
    const [adminCheckError, setAdminCheckError] = useState(null);
    useEffect(() => {
        const verifyAdminStatus = async () => {
            if (!isValidEnvironment || !telegramId)
                return;
            setAdminCheckLoading(true);
            setAdminCheckError(null);
            try {
                const adminStatus = await checkAdminStatus(telegramId);
                setIsAdmin(adminStatus);
                if (!adminStatus) {
                    setAdminCheckError('Access denied: Administrator privileges required');
                }
            }
            catch (error) {
                console.error('Admin verification failed:', error);
                setAdminCheckError('Failed to verify administrator status');
                setIsAdmin(false);
            }
            finally {
                setAdminCheckLoading(false);
            }
        };
        if (isValidEnvironment && telegramId && isAdmin === null) {
            verifyAdminStatus();
        }
    }, [isValidEnvironment, telegramId, isAdmin]);
    const handleRetry = () => {
        window.location.reload();
    };
    // Show loading while checking Telegram environment
    if (isLoading) {
        return (<Container size="sm" pt="xl">
        <Paper p="xl" shadow="sm" style={{ textAlign: 'center' }}>
          <Stack gap="md">
            <Loader size="lg"/>
            <Text size="lg">Initializing Telegram Admin Panel...</Text>
            <Text size="sm" c="dimmed">
              Verifying Telegram WebApp environment
            </Text>
          </Stack>
        </Paper>
      </Container>);
    }
    // Development mode bypass
    if (!isValidEnvironment && shouldUseDevelopmentMode()) {
        devLog('Development mode: bypassing Telegram environment checks');
        return <>{children}</>;
    }
    // Show error if not in valid Telegram environment
    if (!isValidEnvironment) {
        return (<Container size="sm" pt="xl">
        <Paper p="xl" shadow="sm">
          <Stack gap="md">
            <Group justify="center">
              <IconShieldX size={48} color="red"/>
            </Group>
            
            <Text size="xl" fw={600} ta="center" c="red">
              Access Restricted
            </Text>
            
            <Alert icon={<IconAlertTriangle size={16}/>} title="Telegram WebApp Required" color="red">
              This admin panel can only be accessed through Telegram as a Mini App.
              Please open this application from within the Telegram bot.
            </Alert>

            <Stack gap="xs">
              <Text size="sm" c="dimmed">
                <strong>Error:</strong> {error}
              </Text>
              <Text size="sm" c="dimmed">
                <strong>User Agent:</strong> {navigator.userAgent}
              </Text>
              <Text size="sm" c="dimmed">
                <strong>URL:</strong> {window.location.href}
              </Text>
            </Stack>

            <Group justify="center" mt="md">
              <Button leftSection={<IconRefresh size={16}/>} onClick={handleRetry} variant="light">
                Retry
              </Button>
            </Group>
          </Stack>
        </Paper>
      </Container>);
    }
    // Show loading while checking admin status
    if (adminCheckLoading) {
        return (<Container size="sm" pt="xl">
        <Paper p="xl" shadow="sm" style={{ textAlign: 'center' }}>
          <Stack gap="md">
            <IconShield size={48} color="blue" style={{ margin: '0 auto' }}/>
            <Text size="lg">Verifying Administrator Access...</Text>
            <Loader size="md"/>
            <Text size="sm" c="dimmed">
              Checking your permissions with Telegram ID: {telegramId}
            </Text>
          </Stack>
        </Paper>
      </Container>);
    }
    // Show error if admin check failed or user is not admin
    if (isAdmin === false || adminCheckError) {
        return (<Container size="sm" pt="xl">
        <Paper p="xl" shadow="sm">
          <Stack gap="md">
            <Group justify="center">
              <IconShieldX size={48} color="orange"/>
            </Group>
            
            <Text size="xl" fw={600} ta="center" c="orange">
              Insufficient Privileges
            </Text>
            
            <Alert icon={<IconAlertTriangle size={16}/>} title="Administrator Access Required" color="orange">
              You do not have administrator privileges to access this panel.
              Please contact the bot administrator if you believe this is an error.
            </Alert>

            <Stack gap="xs">
              <Group>
                <Text size="sm" c="dimmed">
                  <strong>Telegram ID:</strong>
                </Text>
                <Badge variant="light">{telegramId}</Badge>
              </Group>
              
              {adminCheckError && (<Text size="sm" c="dimmed">
                  <strong>Error:</strong> {adminCheckError}
                </Text>)}
            </Stack>

            <Group justify="center" mt="md">
              <Button leftSection={<IconRefresh size={16}/>} onClick={handleRetry} variant="light">
                Retry
              </Button>
            </Group>
          </Stack>
        </Paper>
      </Container>);
    }
    // All checks passed - render the admin panel
    return <>{children}</>;
};
export default TelegramSecurityGuard;
