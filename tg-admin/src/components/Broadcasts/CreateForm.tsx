import React, { useState, useEffect, useRef } from 'react'
import { Box, Select, Textarea, Button, Group, Title, Alert, SegmentedControl, Text, ActionIcon, Tooltip, FileButton, Image, CloseButton, Stack, Modal, TextInput, SimpleGrid } from '@mantine/core'
import { IconSend, IconInfoCircle, IconUsers, IconUserCheck, IconUserOff, IconBold, IconItalic, IconUnderline, IconStrikethrough, IconCode, IconLink, IconPhoto, IconX } from '@tabler/icons-react'
import { useBroadcasts } from '@/context/BroadcastsContext'
import { notifications } from '@mantine/notifications'
import { useTelegram } from '@/hooks/useTelegram'

const CreateForm: React.FC = () => {
  const { create, creating } = useBroadcasts()
  const { hapticFeedback } = useTelegram()
  const [content, setContent] = useState('')
  const [type, setType] = useState<string>('all')
  const [language, setLanguage] = useState<string | null>(null)
  const [availableLanguages, setAvailableLanguages] = useState<string[]>([])
  const [mediaFile, setMediaFile] = useState<File | null>(null)
  const [mediaPreview, setMediaPreview] = useState<string | null>(null)
  const [linkModalOpen, setLinkModalOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const [linkText, setLinkText] = useState('')
  const [linkSelection, setLinkSelection] = useState({ start: 0, end: 0, text: '' })
  const textareaRef = useRef<HTMLTextAreaElement>(null)

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

  useEffect(() => {
    if (mediaFile) {
      const url = URL.createObjectURL(mediaFile)
      setMediaPreview(url)
      return () => URL.revokeObjectURL(url)
    } else {
      setMediaPreview(null)
    }
  }, [mediaFile])

  const insertTag = (openTag: string, closeTag: string) => {
    const textarea = textareaRef.current
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const selectedText = content.substring(start, end)
    const before = content.substring(0, start)
    const after = content.substring(end)

    const newText = before + openTag + selectedText + closeTag + after
    setContent(newText)

    // Set cursor position after insertion
    setTimeout(() => {
      textarea.focus()
      const newCursorPos = start + openTag.length + selectedText.length + closeTag.length
      textarea.setSelectionRange(
        selectedText ? newCursorPos : start + openTag.length,
        selectedText ? newCursorPos : start + openTag.length
      )
    }, 0)
  }

  const openLinkModal = () => {
    const textarea = textareaRef.current
    if (!textarea) return

    const start = textarea.selectionStart
    const end = textarea.selectionEnd
    const selectedText = content.substring(start, end)
    
    setLinkSelection({ start, end, text: selectedText })
    setLinkUrl('')
    setLinkText(selectedText)
    setLinkModalOpen(true)
  }

  const insertLink = () => {
    if (!linkUrl) return

    const displayText = linkText || linkSelection.text || 'ссылка'
    const before = content.substring(0, linkSelection.start)
    const after = content.substring(linkSelection.end)

    const linkHtml = `<a href="${linkUrl}">${displayText}</a>`
    const newText = before + linkHtml + after
    setContent(newText)
    setLinkModalOpen(false)
    setLinkUrl('')
    setLinkText('')

    setTimeout(() => {
      textareaRef.current?.focus()
    }, 0)
  }

  const formatButtons = [
    { icon: IconBold, tooltip: 'Жирный <b>', action: () => insertTag('<b>', '</b>') },
    { icon: IconItalic, tooltip: 'Курсив <i>', action: () => insertTag('<i>', '</i>') },
    { icon: IconUnderline, tooltip: 'Подчёркнутый <u>', action: () => insertTag('<u>', '</u>') },
    { icon: IconStrikethrough, tooltip: 'Зачёркнутый <s>', action: () => insertTag('<s>', '</s>') },
    { icon: IconCode, tooltip: 'Код <code>', action: () => insertTag('<code>', '</code>') },
    { icon: IconLink, tooltip: 'Ссылка <a>', action: openLinkModal },
  ]

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!content.trim() && !mediaFile) {
      hapticFeedback.error()
      notifications.show({
        title: 'Ошибка',
        message: 'Добавьте текст или медиафайл',
        color: 'red'
      })
      return
    }

    hapticFeedback.soft()
    
    try {
      await create({
        content: content.trim(),
        type,
        language: language || undefined,
        media: mediaFile || undefined
      })
      
      setContent('')
      setLanguage(null)
      setMediaFile(null)
      
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

  const handleMediaSelect = (file: File | null) => {
    if (file) {
      // Check file size (max 50MB for Telegram)
      if (file.size > 50 * 1024 * 1024) {
        notifications.show({
          title: 'Ошибка',
          message: 'Файл слишком большой. Максимум 50MB',
          color: 'red'
        })
        return
      }
      setMediaFile(file)
    }
  }

  const removeMedia = () => {
    setMediaFile(null)
    setMediaPreview(null)
  }

  const topTypeOptions = [
    { value: 'active', label: <Group gap={4} wrap="nowrap"><IconUserCheck size={16} />Активным</Group> },
    { value: 'inactive', label: <Group gap={4} wrap="nowrap"><IconUserOff size={16} />Истекшим</Group> },
    { value: 'no_subscription', label: <Group gap={4} wrap="nowrap"><IconUserOff size={16} />Без подписки</Group> }
  ]

  const isImage = mediaFile?.type.startsWith('image/')
  const isVideo = mediaFile?.type.startsWith('video/')

  return (
    <Box>
      <Title order={3} mb="md">Создать рассылку</Title>
      
      <Alert icon={<IconInfoCircle />} mb="md" variant="light">
        Поддерживается HTML: &lt;b&gt;, &lt;i&gt;, &lt;u&gt;, &lt;s&gt;, &lt;code&gt;, &lt;a href=""&gt;
      </Alert>

      <form onSubmit={handleSubmit}>
        {/* Formatting buttons */}
        <Group gap={4} mb={8}>
          {formatButtons.map(({ icon: Icon, tooltip, action }) => (
            <Tooltip key={tooltip} label={tooltip} position="top">
              <ActionIcon 
                variant="light" 
                size="sm"
                onClick={action}
              >
                <Icon size={14} />
              </ActionIcon>
            </Tooltip>
          ))}
          
          <Box style={{ borderLeft: '1px solid var(--mantine-color-dark-4)', height: 20, margin: '0 4px' }} />
          
          <FileButton
            onChange={handleMediaSelect}
            accept="image/*,video/*"
          >
            {(props) => (
              <Tooltip label="Прикрепить фото/видео" position="top">
                <ActionIcon variant="light" size="sm" {...props}>
                  <IconPhoto size={14} />
                </ActionIcon>
              </Tooltip>
            )}
          </FileButton>
        </Group>

        <Textarea
          ref={textareaRef}
          placeholder="Введите текст рассылки..."
          value={content}
          onChange={(e) => setContent(e.target.value)}
          autosize
          minRows={12}
          mb="md"
          styles={{
            input: {
              fontFamily: 'monospace',
              minHeight: '300px'
            }
          }}
        />

        {/* Media preview */}
        {mediaPreview && (
          <Box mb="md" pos="relative" style={{ display: 'inline-block' }}>
            {isImage && (
              <Image
                src={mediaPreview}
                alt="Preview"
                radius="md"
                maw={200}
                mah={200}
                fit="contain"
              />
            )}
            {isVideo && (
              <video
                src={mediaPreview}
                controls
                style={{ maxWidth: 200, maxHeight: 200, borderRadius: 8 }}
              />
            )}
            <CloseButton
              size="sm"
              pos="absolute"
              top={4}
              right={4}
              onClick={removeMedia}
              style={{ background: 'rgba(0,0,0,0.5)' }}
            />
            <Text size="xs" c="dimmed" mt={4}>
              {mediaFile?.name} ({(mediaFile?.size || 0 / 1024 / 1024).toFixed(2)} MB)
            </Text>
          </Box>
        )}

        <Box mb="md">
          <Text size="sm" fw={500} mb={4}>Получатели</Text>
          <Stack gap="xs">
            <SegmentedControl
              fullWidth
              data={topTypeOptions}
              value={type !== 'all' ? type : ''}
              onChange={(value) => {
                hapticFeedback.selectionChanged()
                setType(value)
              }}
            />
            <SegmentedControl
              fullWidth
              data={[{ value: 'all', label: <Group gap={4} wrap="nowrap"><IconUsers size={16} />Всем</Group> }]}
              value={type === 'all' ? 'all' : ''}
              onChange={(value) => {
                hapticFeedback.selectionChanged()
                setType(value)
              }}
            />
          </Stack>
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
            loading={creating}
            disabled={!content.trim() && !mediaFile}
          >
            {creating ? 'Отправляется...' : 'Отправить рассылку'}
          </Button>
        </Group>
      </form>

      <Modal
        opened={linkModalOpen}
        onClose={() => setLinkModalOpen(false)}
        title="Вставить ссылку"
        centered
        size="sm"
      >
        <TextInput
          label="Название ссылки"
          placeholder="Текст который будет отображаться"
          value={linkText}
          onChange={(e) => setLinkText(e.target.value)}
          mb="md"
          data-autofocus
        />
        <TextInput
          label="URL"
          placeholder="https://example.com"
          value={linkUrl}
          onChange={(e) => setLinkUrl(e.target.value)}
          mb="md"
        />
        <Group justify="flex-end">
          <Button variant="subtle" onClick={() => setLinkModalOpen(false)}>
            Отмена
          </Button>
          <Button onClick={insertLink} disabled={!linkUrl}>
            Вставить
          </Button>
        </Group>
      </Modal>
    </Box>
  )
}

export default CreateForm
