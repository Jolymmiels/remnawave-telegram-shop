import React, { useState, useEffect } from 'react'
import { Box, Select, Textarea, Button, Group, Title, Alert, SegmentedControl, Text } from '@mantine/core'
import { IconSend, IconInfoCircle, IconUsers, IconUserCheck, IconUserOff } from '@tabler/icons-react'
import { useBroadcasts } from '@/context/BroadcastsContext'
import { notifications } from '@mantine/notifications'
import { useTelegram } from '@/hooks/useTelegram'

const CreateForm: React.FC = () => {
  const { create, loading } = useBroadcasts()
  const { hapticFeedback } = useTelegram()
  const [content, setContent] = useState('')
  const [type, setType] = useState<string>('all')
  const [language, setLanguage] = useState<string | null>(null)
  const [availableLanguages, setAvailableLanguages] = useState<string[]>([])

  useEffect(() => {
    const fetchLanguages = async () => {
      try {
        const response = await fetch('/api/languages')
        if (!response.ok) {
          console.error('Failed to fetch languages:', response.status)
          return
        }
        const data = await response.json()
        if (Array.isArray(data)) {
          setAvailableLanguages(data)
        }
      } catch (error) {
        console.error('Failed to fetch languages:', error)
      }
    }
    fetchLanguages()
  }, [])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!content.trim()) {
      hapticFeedback.error()
      notifications.show({
        title: 'Ошибка',
        message: 'Содержание сообщения не может быть пустым',
        color: 'red'
      })
      return
    }

    hapticFeedback.soft()
    
    try {
      await create({
        content: content.trim(),
        type,
        language: language || undefined
      })
      
      setContent('')
      setLanguage(null)
      
      hapticFeedback.success()
      notifications.show({
        title: 'Успешно',
        message: 'Рассылка создана и отправлена',
        color: 'green'
      })
    } catch (error: any) {
      hapticFeedback.error()
      notifications.show({
        title: 'Ошибка',
        message: error.message || 'Не удалось создать рассылку',
        color: 'red'
      })
    }
  }

  const typeOptions = [
    { value: 'all', label: <Group gap={4} wrap="nowrap"><IconUsers size={16} />Всем</Group> },
    { value: 'active', label: <Group gap={4} wrap="nowrap"><IconUserCheck size={16} />Активным</Group> },
    { value: 'inactive', label: <Group gap={4} wrap="nowrap"><IconUserOff size={16} />Неактивным</Group> }
  ]

  return (
    <Box>
      <Title order={3} mb="md">Создать рассылку</Title>
      
      <Alert icon={<IconInfoCircle />} mb="md" variant="light">
        Рассылка будет отправлена немедленно после создания всем выбранным пользователям
      </Alert>

      <form onSubmit={handleSubmit}>
        <Textarea
          label="Содержание сообщения"
          placeholder="Введите текст рассылки..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
          required
          minRows={4}
          maxRows={8}
          mb="md"
        />

        <Box mb="md">
          <Text size="sm" fw={500} mb={4}>Получатели</Text>
          <SegmentedControl
            fullWidth
            data={typeOptions}
            value={type}
            onChange={(value) => {
              hapticFeedback.selectionChanged()
              setType(value)
            }}
          />
        </Box>

        <Select
          label="Язык (необязательно)"
          placeholder="Все языки"
          data={availableLanguages.map(lang => ({ value: lang, label: lang.toUpperCase() }))}
          value={language}
          onChange={setLanguage}
          clearable
          mb="md"
        />

        <Group justify="flex-end">
          <Button
            type="submit"
            leftSection={<IconSend size={16} />}
            loading={loading}
            disabled={!content.trim()}
          >
            {loading ? 'Отправляется...' : 'Отправить рассылку'}
          </Button>
        </Group>
      </form>
    </Box>
  )
}

export default CreateForm