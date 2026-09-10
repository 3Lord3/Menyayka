import { useImperativeHandle, type Ref } from 'react';

import { DeleteOutlined, LockOutlined, PictureOutlined, UploadOutlined } from '@ant-design/icons';
import { App as AntApp, Button, Form, Input, Select, Skeleton, Upload } from 'antd';
import { useNavigate } from 'react-router';

import { getItemImageError, isItemExchanged, ITEM_IMAGE_TYPES } from '@entities/item';

import { categoryOptions, DESCRIPTION_MIN_LENGTH } from '@shared/config/categories';
import { ErrorState } from '@shared/ui';

import { useArchiveItem } from '../model/useArchiveItem';
import { useItemForm } from '../model/useItemForm';

import './ItemForm.scss';

export interface ItemFormHandle {
  confirmLeave: () => void;
}

interface ItemFormProps {
  itemId?: number;
  ref?: Ref<ItemFormHandle>;
}

export function ItemForm({ itemId, ref }: ItemFormProps) {
  const navigate = useNavigate();
  const { message, modal } = AntApp.useApp();
  const archiveItem = useArchiveItem(() => navigate('/products'));
  const {
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
  } = useItemForm(itemId);

  useImperativeHandle(ref, () => ({ confirmLeave }), [confirmLeave]);

  if (isLoading) {
    return (
      <div className="item-form__skeleton">
        <Skeleton active paragraph={{ rows: 4 }} />
      </div>
    );
  }

  if (isLoadError) {
    return <ErrorState onRetry={() => window.location.reload()} />;
  }

  function confirmArchive() {
    modal.confirm({
      title: 'Удалить товар?',
      content: `«${item?.title ?? 'Товар'}» будет удалён из списка товаров.`,
      okText: 'Да, удалить',
      okButtonProps: { danger: true },
      cancelText: 'Отмена',
      onOk: () => archiveItem.mutate(itemId as number),
    });
  }

  // the backend rejects any mutation of an archived item, and you can only reach this screen
  // via a direct link
  const exchanged = Boolean(item && isItemExchanged(item.status));

  return (
    <Form
      className="item-form"
      form={form}
      layout="vertical"
      name="item-form"
      initialValues={initialValues}
      disabled={submitting || exchanged}
      onValuesChange={handleValuesChange}
      onFinish={handleSubmit}
    >
      {exchanged ? (
        <div className="item-form__lock" role="status">
          <LockOutlined className="item-form__lock-icon" aria-hidden />
          <span>Товар обменян и больше не редактируется</span>
        </div>
      ) : null}
      <Form.Item label="Фото" required>
        <div className="item-form__photo">
          {hasPhoto ? (
            <div className="item-form__photo-preview item-form__photo-preview--filled">
              {previewUrl ? (
                <img className="item-form__photo-image" src={previewUrl} alt="Фото товара" />
              ) : (
                // a file is chosen but no preview is available (no Object URL API) — a
                // placeholder instead of an <img> without src, which renders as a broken image
                <>
                  <PictureOutlined className="item-form__photo-add-icon" aria-hidden />
                  <span className="item-form__photo-add">Фото выбрано</span>
                </>
              )}
            </div>
          ) : (
            <Upload
              className="item-form__photo-upload"
              accept={ITEM_IMAGE_TYPES.join(',')}
              showUploadList={false}
              beforeUpload={(file) => {
                const imageError = getItemImageError(file);
                if (imageError) {
                  message.error(imageError);
                  return Upload.LIST_IGNORE;
                }
                handleImageSelected(file);
                return Upload.LIST_IGNORE;
              }}
            >
              <div className="item-form__photo-preview item-form__photo-preview--empty">
                <UploadOutlined className="item-form__photo-add-icon" aria-hidden />
                <span className="item-form__photo-add">Добавить фото</span>
              </div>
            </Upload>
          )}
          {hasPhoto ? (
            <Button
              className="item-form__photo-remove"
              type="text"
              danger
              icon={<DeleteOutlined aria-hidden />}
              onClick={handleImageRemove}
            >
              Удалить фото
            </Button>
          ) : null}
        </div>
      </Form.Item>
      <Form.Item
        label="Название"
        name="title"
        rules={[
          { required: true, message: 'Введите название' },
          { max: 100, message: 'Название не длиннее 100 символов' },
        ]}
      >
        <Input placeholder="Название товара" maxLength={100} showCount />
      </Form.Item>
      {/* selection only from the dictionary: clustering compares categories by exact string
    equality, so free text would move the request into a one-item pool */}
      <Form.Item
        label="Категория"
        name="category"
        rules={[{ required: true, message: 'Выберите категорию' }]}
      >
        <Select
          placeholder="Выберите категорию"
          options={categoryOptions(item?.category)}
          showSearch
        />
      </Form.Item>
      <Form.Item
        label="Описание"
        name="description"
        rules={[
          { required: true, message: 'Введите описание' },
          { max: 500, message: 'Описание не длиннее 500 символов' },
          {
            // a custom validator instead of min: antd counts pre-truncation characters, while
            // values.description.trim() goes to the backend — 25 spaces would pass min
            validator(_, value: string | undefined) {
              if (!value || value.trim().length >= DESCRIPTION_MIN_LENGTH) {
                return Promise.resolve();
              }
              return Promise.reject(new Error('Пожалуйста, опишите товар подробнее'));
            },
          },
        ]}
      >
        <Input.TextArea placeholder="Описание товара" maxLength={500} showCount rows={4} />
      </Form.Item>
      <Button
        className="item-form__submit"
        type="primary"
        htmlType="submit"
        size="large"
        block
        loading={submitting}
        // an explicit disabled on the button overrides the form's disabled (antd merges as
        // customDisabled ?? contextDisabled) — we add the exchanged item here ourselves
        disabled={!canSubmit || exchanged}
      >
        {isEdit ? 'Сохранить изменения' : 'Сохранить'}
      </Button>
      {isEdit && !exchanged ? (
        <Button
          className="item-form__archive"
          danger
          block
          loading={archiveItem.isPending}
          disabled={submitting}
          onClick={confirmArchive}
        >
          Удалить товар
        </Button>
      ) : null}
    </Form>
  );
}
