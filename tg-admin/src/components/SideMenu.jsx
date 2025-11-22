import React from 'react';
import { NavLink, Stack, Box } from '@mantine/core';
import { IconChartBar, IconUsers, IconSpeakerphone, IconTicket } from '@tabler/icons-react';
import { useTelegram } from '@/hooks/useTelegram';
const SideMenu = ({ onSelect }) => {
    const { hapticFeedback } = useTelegram();
    const menuItems = [
        { path: '/purchases', label: 'Статистика покупок', icon: IconChartBar },
        { path: '/users', label: 'Статистика пользователей', icon: IconUsers },
        { path: '/broadcasts', label: 'Вещание', icon: IconSpeakerphone },
        { path: '/promos', label: 'Промокоды', icon: IconTicket }
    ];
    const handleNavigation = (path) => {
        hapticFeedback.soft();
        onSelect(path);
    };
    return (<Box p="md">
      <Stack gap="xs">
        {menuItems.map((item) => (<NavLink key={item.path} label={item.label} leftSection={<item.icon size="1rem"/>} onClick={() => handleNavigation(item.path)} style={{ cursor: 'pointer' }}/>))}
      </Stack>
    </Box>);
};
export default SideMenu;
