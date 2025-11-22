import React, { useState } from 'react'
import {
  Card,
  TextInput,
  NumberInput,
  Button,
  Group,
  Stack,
  Text,
  Switch,
  Alert
} from '@mantine/core'
import { useForm } from '@mantine/form'
import { notifications } from '@mantine/notifications'
import { IconCheck, IconX } from '@tabler/icons-react'
import { usePromos, CreatePromoData } from '@/context/PromosContext'
import {useTelegram} from "@/hooks/useTelegram";

const CreatePromoForm: React.FC = () => {
  const { create } = usePromos()
  const { hapticFeedback } = useTelegram()
  const [loading, setLoading] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)

  const form = useForm<CreatePromoData & { hasMaxUses: boolean; hasExpiration: boolean; expirationDate: string }>({
    initialValues: {
      code: '',
      bonus_days: 7,
      hasMaxUses: false,
      max_uses: 100,
      hasExpiration: false,
      expires_at: undefined,
      expirationDate: ''
    },
    validate: {
      code: (value: string) => {
        if (!value || value.trim().length === 0) return 'Необходим код'
        if (!/^[A-Z0-9_-]+$/i.test(value)) return 'Код может содержать только буквы, цифры, символы подчеркивания и дефисы.'
        return null
      },
      bonus_days: (value: number) => (value && value > 0) ? null : 'Бонусные дни должны быть больше 0',
      max_uses: (value: number | undefined, values: any) => {
        if (values.hasMaxUses && (!value || value <= 0)) {
          return 'Максимальное количество использований должно быть больше 0'
        }
        return null
      }
    }
  })

  const handleSubmit = async (values: typeof form.values) => {
    setLoading(true)
    hapticFeedback.soft()
    try {
      const data: CreatePromoData = {
        code: values.code.trim().toUpperCase(),
        bonus_days: values.bonus_days,
        max_uses: values.hasMaxUses ? values.max_uses : undefined,
        expires_at: values.hasExpiration && values.expirationDate 
          ? new Date(values.expirationDate).toISOString() 
          : undefined
      }
      
      await create(data)
      form.reset()
      setShowAdvanced(false)
      notifications.show({
        title: 'Успех',
        message: 'Промокод успешно создан',
        color: 'green',
        icon: <IconCheck size={16} />
      })
    } catch (error) {
      notifications.show({
        title: 'Error',
        message: error instanceof Error ? error.message : 'Не удалось создать промокод',
        color: 'red',
        icon: <IconX size={16} />
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card shadow="sm" padding="lg" radius="md" withBorder mb="xl">
      <form onSubmit={form.onSubmit(handleSubmit)}>
        <Stack>
          <Group grow>
            <TextInput
              label="Промокод"
              placeholder="SAVE20"
              required
              {...form.getInputProps('code')}
              styles={{ input: { textTransform: 'uppercase' } }}
            />
            <NumberInput
              label="Бонусные дни"
              placeholder="7"
              min={1}
              required
              {...form.getInputProps('bonus_days')}
            />
          </Group>

          <Switch
            label="Показать дополнительные параметры"
            checked={showAdvanced}
            onChange={(event) => {
              hapticFeedback.soft()
              setShowAdvanced(event.currentTarget.checked)
            }}
          />

          {showAdvanced && (
            <Stack>
              <Switch
                label="Ограничить количество использований"
                {...form.getInputProps('hasMaxUses', { type: 'checkbox' })}
                onChange={(event) => {
                  hapticFeedback.soft()
                  form.setFieldValue('hasMaxUses', event.currentTarget.checked)
                }}
              />

              {form.values.hasMaxUses && (
                <NumberInput
                  label="Максимальное кол-во использований"
                  placeholder="100"
                  min={1}
                  {...form.getInputProps('max_uses')}
                />
              )}

              <Switch
                label="Установить дату истечения срока действия"
                {...form.getInputProps('hasExpiration', { type: 'checkbox' })}
                onChange={(event) => {
                  hapticFeedback.soft()
                  form.setFieldValue('hasExpiration', event.currentTarget.checked)
                }}
              />

              {form.values.hasExpiration && (
                <TextInput
                  label="Срок действия истекает до"
                  type="datetime-local"
                  description="Выберите дату и время истечения промокода"
                  min={new Date().toISOString().slice(0, 16)}
                  {...form.getInputProps('expirationDate')}
                />
              )}
            </Stack>
          )}

          <Group>
            <Button type="submit" loading={loading}>
              Создать
            </Button>
            <Button variant="subtle" onClick={() => {
              hapticFeedback.soft()
              form.reset()
            }}>
              Сбросить
            </Button>
          </Group>
        </Stack>
      </form>
    </Card>
  )
}

export default CreatePromoForm