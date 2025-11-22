import React, { useState } from 'react';
import { Box, TextInput, Textarea, Select, Button, Group, Title, Alert } from '@mantine/core';
import { IconSend, IconInfoCircle } from '@tabler/icons-react';
import { useBroadcasts } from '@/context/BroadcastsContext';
import { notifications } from '@mantine/notifications';
import { useTelegram } from '@/hooks/useTelegram';
const CreateForm = () => {
    const { create, loading } = useBroadcasts();
    const { hapticFeedback } = useTelegram();
    const [content, setContent] = useState('');
    const [type, setType] = useState('all');
    const [language, setLanguage] = useState('');
    const handleSubmit = async (e) => {
        e.preventDefault();
        if (!content.trim()) {
            hapticFeedback.error();
            notifications.show({
                title: 'Ошибка',
                message: 'Содержание сообщения не может быть пустым',
                color: 'red'
            });
            return;
        }
        hapticFeedback.soft();
        try {
            await create({
                content: content.trim(),
                type,
                language: language.trim() || undefined
            });
            setContent('');
            setLanguage('');
            hapticFeedback.success();
            notifications.show({
                title: 'Успешно',
                message: 'Рассылка создана и отправлена',
                color: 'green'
            });
        }
        catch (error) {
            hapticFeedback.error();
            notifications.show({
                title: 'Ошибка',
                message: error.message || 'Не удалось создать рассылку',
                color: 'red'
            });
        }
    };
    const handleSelectionChange = () => {
        hapticFeedback.selectionChanged();
    };
    const typeOptions = [
        { value: 'all', label: '🌍 Всем пользователям' },
        { value: 'active', label: '✅ Только активным' },
        { value: 'inactive', label: '⏰ Только неактивным' }
    ];
    return (<Box>
      <Title order={3} mb="md">Создать рассылку</Title>
      
      <Alert icon={<IconInfoCircle />} mb="md" variant="light">
        Рассылка будет отправлена немедленно после создания всем выбранным пользователям
      </Alert>

      <form onSubmit={handleSubmit}>
        <Textarea label="Содержание сообщения" placeholder="Введите текст рассылки..." value={content} onChange={(e) => setContent(e.target.value)} required minRows={4} maxRows={8} mb="md"/>

        <Group grow mb="md">
          <Select label="Тип получателей" data={typeOptions} value={type} onChange={(value) => {
            handleSelectionChange();
            setType(value || 'all');
        }} allowDeselect={false}/>
          
          <TextInput label="Язык (необязательно)" placeholder="ru, en, es..." value={language} onChange={(e) => setLanguage(e.target.value)}/>
        </Group>

        <Group justify="flex-end">
          <Button type="submit" leftSection={<IconSend size={16}/>} loading={loading} disabled={!content.trim()}>
            {loading ? 'Отправляется...' : 'Отправить рассылку'}
          </Button>
        </Group>
      </form>
    </Box>);
};
export default CreateForm;
