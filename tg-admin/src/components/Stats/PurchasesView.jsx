import React from 'react';
import { Text, Paper, Grid, Stack } from '@mantine/core';
const PurchasesView = () => {
    return (<Stack>
      <Grid>
        <Grid.Col span={12}>
          <Paper p="md" shadow="sm">
            <Text size="lg" fw={500} mb="sm">Общая статистика</Text>
            <Text c="dimmed">
              Здесь будет отображаться статистика покупок
            </Text>
          </Paper>
        </Grid.Col>
      </Grid>
    </Stack>);
};
export default PurchasesView;
