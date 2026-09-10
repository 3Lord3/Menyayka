import { useRef } from 'react';

import { ArrowLeftOutlined } from '@ant-design/icons';
import { Button } from 'antd';
import { useNavigate, useParams } from 'react-router';

import { ItemForm, type ItemFormHandle } from '@features/items';

import { ErrorState } from '@shared/ui';

import './ItemFormPage.scss';

export function ItemFormPage() {
  const { itemId } = useParams();
  const navigate = useNavigate();
  const formRef = useRef<ItemFormHandle>(null);

  const numericItemId = itemId ? Number(itemId) : undefined;

  // a non-numeric id from the address bar (handwritten link) is an invalid URL: we don't open
  // the edit form, we lead to the list where a valid item can be chosen
  if (itemId && !Number.isInteger(numericItemId)) {
    return (
      <div className="item-form-page">
        <div className="item-form-page__body">
          <ErrorState onRetry={() => navigate('/products')} />
        </div>
      </div>
    );
  }

  return (
    <div className="item-form-page">
      <header className="item-form-page__header">
        <Button
          className="item-form-page__back"
          type="text"
          icon={<ArrowLeftOutlined aria-hidden />}
          aria-label="Назад"
          onClick={() => formRef.current?.confirmLeave()}
        />
        <h1 className="item-form-page__title">
          {itemId ? 'Редактирование товара' : 'Новый товар'}
        </h1>
      </header>
      <div className="item-form-page__body">
        <ItemForm itemId={numericItemId} ref={formRef} />
      </div>
    </div>
  );
}
