import { useEffect, useMemo, useState } from 'react';

import { useQueryClient } from '@tanstack/react-query';
import { App as AntApp, Form, type UploadProps } from 'antd';
import { useNavigate, useSearchParams } from 'react-router';

import {
  createItem,
  ItemImageUploadError,
  updateItem,
  useItem,
  type ItemPayload,
  type ItemsList,
} from '@entities/item';

import { DESCRIPTION_MIN_LENGTH } from '@shared/config/categories';
import { getErrorMessage } from '@shared/lib/errorMessage';

type UploadedFile = Parameters<NonNullable<UploadProps['beforeUpload']>>[0];

export interface ItemFormValues {
  title: string;
  description: string;
  category?: string;
}

export function useItemForm(itemId?: number) {
  const { message, modal } = AntApp.useApp();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const [form] = Form.useForm<ItemFormValues>();
  const itemQuery = useItem(itemId);
  const item = itemQuery.data;

  const [pendingFile, setPendingFile] = useState<UploadedFile | null>(null);
  const [removingImage, setRemovingImage] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [dirty, setDirty] = useState(false);

  const isEdit = Boolean(itemId);
  // creating an item from the request form: after saving we return to it with the new item
  // selected
  const returnToRequest = searchParams.get('returnTo') === 'request';
  const isLoading = isEdit && itemQuery.isPending;
  const isLoadError = isEdit && itemQuery.isError;

  const initialValues: Partial<ItemFormValues> | undefined = useMemo(() => {
    if (!item) return undefined;
    return {
      title: item.title,
      description: item.description,
      // a Select with value='' draws an empty chip instead of a placeholder
      category: item.category || undefined,
    };
  }, [item]);

  // the photo is required in both forms: removing a photo from an existing item makes it
  // unsaveable
  const hasPhoto = Boolean(pendingFile) || (isEdit && Boolean(item?.imageUrl) && !removingImage);

  // a blob URL for the chosen file lives exactly until the file changes or unmount —
  // otherwise every new photo (up to 5 MB) stays hanging in the tab's memory
  const pendingPreview = useMemo(
    () =>
      pendingFile && typeof URL.createObjectURL === 'function'
        ? URL.createObjectURL(pendingFile)
        : null,
    [pendingFile],
  );
  useEffect(() => {
    if (!pendingPreview) return;
    return () => URL.revokeObjectURL(pendingPreview);
  }, [pendingPreview]);

  // showing the old photo instead of the freshly chosen one is dishonest
  const previewUrl = pendingFile
    ? pendingPreview
    : isEdit && item?.imageUrl && !removingImage
      ? item.imageUrl
      : null;

  const title = Form.useWatch('title', form);
  const description = Form.useWatch('description', form);
  const category = Form.useWatch('category', form);

  useEffect(() => {
    if (!item) return;
    form.validateFields(['description']).catch(() => undefined);
  }, [form, item]);

  const descriptionLength = description?.trim().length ?? 0;
  const fieldsValid =
    Boolean(title?.trim()) && descriptionLength >= DESCRIPTION_MIN_LENGTH && Boolean(category);
  const canSubmit = fieldsValid && hasPhoto && !submitting;

  function handleImageSelected(file: UploadedFile) {
    setPendingFile(file);
    setRemovingImage(false);
    setDirty(true);
  }

  function handleImageRemove() {
    setPendingFile(null);
    if (item?.imageUrl) setRemovingImage(true);
    setDirty(true);
  }

  function handleValuesChange() {
    setDirty(true);
  }

  function goBack() {
    navigate(returnToRequest ? '/exchange-requests/new' : '/products');
  }

  function confirmLeave() {
    if (!dirty) {
      goBack();
      return;
    }
    // the form isn't complete — nothing to save, we only offer to leave or stay
    if (!canSubmit) {
      modal.confirm({
        title: 'Изменения не сохранены',
        content: 'Форма заполнена не полностью — сохранить нельзя. Выйти без сохранения?',
        okText: 'Выйти без сохранения',
        okButtonProps: { danger: true },
        cancelText: 'Остаться',
        closable: false,
        centered: true,
        mask: { closable: false },
        onOk: () => goBack(),
      });
      return;
    }
    modal.confirm({
      title: 'Изменения не сохранены',
      content: 'Хотите сохранить изменения или вернуться назад?',
      okText: 'Сохранить изменения',
      cancelText: 'Назад',
      closable: false,
      centered: true,
      mask: { closable: false },
      // form.submit()/validateFields() from the modal callback don't finish in this context —
      // we take the values synchronously and send them ourselves (we get here only when canSubmit)
      onOk: () => {
        handleSubmit(form.getFieldsValue() as ItemFormValues);
      },
      onCancel: () => goBack(),
    });
  }

  async function handleSubmit(values: ItemFormValues) {
    if (submitting) return;
    setSubmitting(true);
    try {
      const payload: ItemPayload = {
        title: values.title.trim(),
        description: values.description.trim(),
        // ?? '' guards against a PATCH without the key: it would leave the old server value
        category: values.category ?? '',
      };
      if (isEdit) {
        await updateItem(itemId as number, payload, pendingFile ?? undefined);
        message.success('Товар обновлён');
        queryClient.invalidateQueries({ queryKey: ['items'] });
        // request cards show the item name — otherwise it stays stale there for another minute
        queryClient.invalidateQueries({ queryKey: ['exchange-requests'] });
        navigate(returnToRequest ? `/exchange-requests/new?offeredItemId=${itemId}` : '/products');
      } else {
        const created = await createItem(payload, pendingFile);
        message.success('Товар создан');
        queryClient.invalidateQueries({ queryKey: ['items'] });
        // the request form reads the cache synchronously on mount — the new item must already
        // be there, otherwise the offeredItemId preset won't apply until a background refetch
        queryClient.setQueryData<ItemsList>(['items'], (old) => ({
          items: old ? [...old.items, created] : [created],
          total: (old?.total ?? 0) + 1,
        }));
        if (returnToRequest) {
          navigate(`/exchange-requests/new?offeredItemId=${created.id}`);
        } else {
          navigate('/products');
        }
      }
    } catch (error) {
      // the item is already saved but the photo failed — don't lose the changes, return to
      // the form
      if (error instanceof ItemImageUploadError) {
        message.error(
          isEdit
            ? 'Товар обновлён, но фото не загрузилось — попробуйте загрузить его ещё раз'
            : 'Товар создан, но фото не загрузилось. Добавьте его на экране редактирования',
        );
        queryClient.invalidateQueries({ queryKey: ['items'] });
        queryClient.invalidateQueries({ queryKey: ['exchange-requests'] });
        if (!isEdit) {
          navigate(
            returnToRequest
              ? `/products/${error.item.id}/edit?returnTo=request`
              : `/products/${error.item.id}/edit`,
          );
          return;
        }
        setSubmitting(false);
        return;
      }
      message.error(
        getErrorMessage(
          error,
          { 409: 'Товар уже участвует в сделке' },
          'Не удалось сохранить товар',
        ),
      );
      setSubmitting(false);
    }
  }

  return {
    form,
    item,
    isEdit,
    isLoading,
    isLoadError,
    submitting,
    canSubmit,
    initialValues,
    previewUrl,
    hasPhoto,
    handleImageSelected,
    handleImageRemove,
    handleValuesChange,
    confirmLeave,
    handleSubmit,
  };
}
