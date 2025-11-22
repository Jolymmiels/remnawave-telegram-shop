import React from 'react'
import { Title, Text, Paper, Grid, Stack } from '@mantine/core'

const PurchasesView: React.FC = () => {
  return (
    <Stack>
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
    </Stack>
  )
}

export default PurchasesView