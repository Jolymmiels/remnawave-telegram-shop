import React from 'react';
import { Box } from '@mantine/core';
const AppHeader = ({ title }) => {
    return (<Box h="100%" px="md" style={{
            paddingTop: 'max(var(--mantine-spacing-sm), var(--tg-viewport-safe-area-inset-top, 0px))'
        }}>
    </Box>);
};
export default AppHeader;
