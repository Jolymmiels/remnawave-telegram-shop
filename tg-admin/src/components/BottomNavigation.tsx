import React from 'react'
import { Group, UnstyledButton, Text, Stack } from '@mantine/core'
import { useLocation } from 'react-router-dom'
import { IconChartBar, IconUsers, IconSpeakerphone, IconTicket } from '@tabler/icons-react'
import { useTelegram } from '@/hooks/useTelegram'

interface BottomNavigationProps {
  onSelect: (path: string) => void
}

const BottomNavigation: React.FC<BottomNavigationProps> = ({ onSelect }) => {
  const location = useLocation()
  const { hapticFeedback } = useTelegram()

  const navigationItems = [
    {
      path: '/purchases',
      label: 'Статистика',
      icon: IconChartBar
    },
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
  ]

  const handleNavigation = (path: string) => {
    hapticFeedback.soft()
    onSelect(path)
  }

  return (
    <Group 
      justify="space-around" 
      h="100%" 
      px="xs"
      wrap="nowrap"
      className="no-select"
      style={{
        backgroundColor: 'var(--mantine-color-dark-7)'
      }}
    >
      {navigationItems.map(({ path, label, icon: Icon }) => {
        const isActive = location.pathname === path
        
        return (
          <UnstyledButton
            key={path}
            onClick={() => handleNavigation(path)}
            style={{
              padding: '4px 6px',
              borderRadius: '8px',
              textAlign: 'center',
              flex: 1,
              opacity: isActive ? 1 : 0.6,
              color: isActive 
                ? 'var(--tg-theme-link-color, var(--mantine-color-blue-6))' 
                : 'var(--tg-theme-text-color, var(--mantine-color-dark-7))'
            }}
          >
            <Stack gap={2} align="center">
              <Icon size={18} stroke={1.5} />
              <Text size="10px" fw={500} style={{ whiteSpace: 'nowrap' }}>
                {label}
              </Text>
            </Stack>
          </UnstyledButton>
        )
      })}
    </Group>
  )
}

export default BottomNavigation