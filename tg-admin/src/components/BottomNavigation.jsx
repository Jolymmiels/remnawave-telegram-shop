import React from 'react';
import { Group, UnstyledButton, Text, Stack } from '@mantine/core';
import { useLocation } from 'react-router-dom';
import { IconUsers, IconSpeakerphone, IconTicket } from '@tabler/icons-react';
import { useTelegram } from '@/hooks/useTelegram';
const BottomNavigation = ({ onSelect }) => {
    const location = useLocation();
    const { hapticFeedback } = useTelegram();
    const navigationItems = [
        {
            path: '/users',
            label: 'Пользователи',
            icon: IconUsers
        },
        {
            path: '/broadcasts',
            label: 'Рассылка',
            icon: IconSpeakerphone
        },
        {
            path: '/promos',
            label: 'Промокоды',
            icon: IconTicket
        }
    ];
    const handleNavigation = (path) => {
        hapticFeedback.soft();
        onSelect(path);
    };
    return (<Group justify="space-around" h="100%" px="xs" className="no-select" style={{
            backgroundColor: 'var(--mantine-color-dark-7)'
        }}>
      {navigationItems.map(({ path, label, icon: Icon }) => {
            const isActive = location.pathname === path;
            return (<UnstyledButton key={path} onClick={() => handleNavigation(path)} style={{
                    padding: '8px 12px',
                    borderRadius: '8px',
                    textAlign: 'center',
                    minWidth: '60px',
                    opacity: isActive ? 1 : 0.6,
                    color: isActive
                        ? 'var(--tg-theme-link-color, var(--mantine-color-blue-6))'
                        : 'var(--tg-theme-text-color, var(--mantine-color-dark-7))'
                }}>
            <Stack gap={4} align="center">
              <Icon size={20} stroke={1.5}/>
              <Text size="xs" fw={isActive ? 600 : 400}>
                {label}
              </Text>
            </Stack>
          </UnstyledButton>);
        })}
    </Group>);
};
export default BottomNavigation;
