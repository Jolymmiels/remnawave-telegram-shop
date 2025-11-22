import React, { useState, useEffect } from 'react'
import {
  Modal,
  Stack,
  TextInput,
  Select,
  Button,
  Group,
} from '@mantine/core'
import { http } from '../../lib/http'
import { notifications } from '@mantine/notifications'
import { getTelegramSafeAreaStyles } from '../../lib/telegram-safe-area'

interface User {
  id: number
  telegram_id: number
  expire_at?: string
  created_at: string
  subscription_link?: string
  language: string
  payments_count: number
  referrals_count: number
  total_spent: number
}

interface UserEditModalProps {
  user: User | null
  opened: boolean
  onClose: () => void
  onUserUpdated: () => void
}

const UserEditModal: React.FC<UserEditModalProps> = ({
  user,
  opened,
  onClose,
  onUserUpdated,
}) => {
  const [loading, setLoading] = useState(false)
  const [language, setLanguage] = useState('')
  const [subscriptionLink, setSubscriptionLink] = useState('')
  const [expireAtDate, setExpireAtDate] = useState('')

  // Helper function to convert ISO date to datetime-local format
  const formatDateForInput = (isoDateString: string | null | undefined): string => {
    if (!isoDateString) return ''
    const date = new Date(isoDateString)
    // Format to local datetime string for input (YYYY-MM-DDTHH:MM)
    return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
      .toISOString()
      .slice(0, 16)
  }

  // Helper function to reset form values
  const resetForm = () => {
    if (user) {
      setLanguage(user.language)
      setSubscriptionLink(user.subscription_link || '')
      setExpireAtDate(formatDateForInput(user.expire_at))
    }
  }

  useEffect(() => {
    resetForm()
  }, [user])

  const handleSave = async () => {
    if (!user) return

    setLoading(true)
    try {
      const updateData: any = {
        language,
        subscription_link: subscriptionLink || null,
      }

      if (expireAtDate) {
        // Convert datetime-local value back to ISO string
        updateData.expire_at = new Date(expireAtDate).toISOString()
      } else {
        updateData.expire_at = null
      }

      await http.put(`/api/users/${user.telegram_id}/update`, updateData)
      
      notifications.show({
        title: 'Успешно',
        message: 'Пользователь обновлен',
        color: 'green',
      })
      
      onUserUpdated()
      onClose()
    } catch (error) {
      console.error('Failed to update user:', error)
      notifications.show({
        title: 'Ошибка',
        message: 'Не удалось обновить пользователя',
        color: 'red',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    resetForm()
    onClose()
  }

  if (!user) return null

  return (
    <Modal 
      opened={opened} 
      onClose={handleClose} 
      title={`Редактирование пользователя ${user.telegram_id}`}
      styles={getTelegramSafeAreaStyles()}
    >
      <Stack>
        <TextInput
          label="Telegram ID"
          value={user.telegram_id}
          disabled
          description="Telegram ID нельзя изменить"
        />
        
        <Select
          label="Язык"
          value={language}
          onChange={(value) => setLanguage(value || '')}
          data={[
            { value: 'ru', label: 'Русский' },
            { value: 'en', label: 'English' },
          ]}
          required
        />

        <TextInput
          label="Ссылка на подписку"
          value={subscriptionLink}
          onChange={(e) => setSubscriptionLink(e.currentTarget.value)}
          placeholder="https://..."
          description="Ссылка на подписку пользователя"
        />

        <TextInput
          label="Подписка истекает"
          type="datetime-local"
          value={expireAtDate}
          onChange={(e) => setExpireAtDate(e.currentTarget.value)}
          placeholder="Выберите дату и время"
          description="Оставьте пустым для бессрочной подписки"
        />

        <Group justify="flex-end" mt="md">
          <Button variant="default" onClick={handleClose}>
            Отмена
          </Button>
          <Button onClick={handleSave} loading={loading}>
            Сохранить
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}

export default UserEditModal