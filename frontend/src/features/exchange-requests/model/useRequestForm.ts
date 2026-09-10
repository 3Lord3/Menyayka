import { useEffect, useState } from 'react';

import { useQueryClient } from '@tanstack/react-query';
import { App as AntApp, Form } from 'antd';
import { isAxiosError } from 'axios';
import { useNavigate, useSearchParams } from 'react-router';

import { fetchExchangeOptions, invalidateChainQueries } from '@entities/chain';
import {
  createRequest,
  isRequestEditable,
  updateRequest,
  useRequest,
  type CreateRequestResult,
} from '@entities/exchangeRequest';
import { useItems } from '@entities/item';

import { DESCRIPTION_MIN_LENGTH } from '@shared/config/categories';
import { getErrorMessage } from '@shared/lib/errorMessage';

export interface RequestFormValues {
  offeredItemId: number;
  wantedDescription: string;
  wantedCategory?: string;
}

export function useRequestForm(requestId?: number) {
  const { message, modal } = AntApp.useApp();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const [form] = Form.useForm<RequestFormValues>();
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<CreateRequestResult | null>(null);
  const [dirty, setDirty] = useState(false);

  const requestQuery = useRequest(requestId);
  const request = requestQuery.data;
  const itemsQuery = useItems();

  const isEdit = Boolean(requestId);
  const items = itemsQuery.data?.items ?? [];
  const readOnly = isEdit && request ? !isRequestEditable(request.status) : false;
  const readOnlyReason = readOnly && request ? request.status : null;
  const isLoading = isEdit ? requestQuery.isPending : itemsQuery.isPending;
  const isLoadError = isEdit ? requestQuery.isError : itemsQuery.isError;

  const preselectedItemId = searchParams.get('offeredItemId');

  // the form mounts only after the data load, so initialValues suffice
  const initialValues: Partial<RequestFormValues> | undefined = isEdit
    ? request
      ? {
          offeredItemId: request.offeredItemId,
          wantedDescription: request.wantedDescription,
          wantedCategory: request.wantedCategory || undefined,
        }
      : undefined
    : {
        offeredItemId:
          preselectedItemId &&
          items.some((item) => item.id === Number(preselectedItemId) && item.status === 'ACTIVE')
            ? Number(preselectedItemId)
            : undefined,
      };

  const offeredItemId = Form.useWatch('offeredItemId', form);
  const wantedDescription = Form.useWatch('wantedDescription', form);
  const wantedCategory = Form.useWatch('wantedCategory', form);

  useEffect(() => {
    if (!request || readOnly) return;
    form.validateFields(['wantedDescription']).catch(() => undefined);
  }, [form, request, readOnly]);

  const canSubmit =
    (wantedDescription?.trim().length ?? 0) >= DESCRIPTION_MIN_LENGTH &&
    Boolean(wantedCategory) &&
    (isEdit || Boolean(offeredItemId)) &&
    !submitting &&
    !readOnly;

  async function handleSubmit(values: RequestFormValues) {
    if (submitting) return;
    setSubmitting(true);
    try {
      if (isEdit) {
        if (!request) return;
        await updateRequest(requestId as number, {
          offeredItemId: request.offeredItemId,
          wantedDescription: values.wantedDescription.trim(),
          // required field; ?? '' is a guard — PUT overwrites the request entirely
          wantedCategory: values.wantedCategory ?? '',
          version: request.version,
        });
        message.success('Запрос обновлён');
        // editing a request recomputes candidate chains on the server
        invalidateChainQueries(queryClient);
        navigate('/exchange-requests');
      } else {
        const created = await createRequest({
          offeredItemId: values.offeredItemId,
          wantedDescription: values.wantedDescription.trim(),
          wantedCategory: values.wantedCategory ?? '',
        });
        // matching on the backend is synchronous, so the chains are already in the DB by
        // response time — their count comes from exchange-options
        const options = await fetchExchangeOptions(created.id).catch(() => []);
        setResult({ request: created, matching: { createdCandidateChains: options.length } });
        invalidateChainQueries(queryClient);
      }
      setDirty(false);
    } catch (error) {
      // version conflict (409) — the request changed in another window; the cache is stale,
      // not the form
      if (isEdit && isAxiosError(error) && error.response?.status === 409) {
        message.error('Заявка изменилась — обновите страницу и попробуйте ещё раз');
        queryClient.invalidateQueries({ queryKey: ['exchange-requests'] });
        return;
      }
      message.error(
        getErrorMessage(
          error,
          { 400: 'Проверьте заполнение полей', 409: 'Товар уже в резерве' },
          'Не удалось сохранить запрос',
        ),
      );
    } finally {
      // including on early exit before submitting: otherwise the form stays blocked forever
      setSubmitting(false);
    }
  }

  function goToList() {
    navigate('/exchange-requests');
  }

  function goToChains() {
    if (!result) return;
    navigate(`/exchange-requests/${result.request.id}`);
  }

  function goCreateItem() {
    navigate('/products/new?returnTo=request');
  }

  function handleValuesChange() {
    setDirty(true);
  }

  function confirmLeave() {
    if (!dirty) {
      goToList();
      return;
    }
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
        onOk: () => goToList(),
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
      onOk: () => {
        handleSubmit(form.getFieldsValue() as RequestFormValues);
      },
      onCancel: () => goToList(),
    });
  }

  return {
    form,
    request,
    isEdit,
    isLoading,
    isLoadError,
    submitting,
    canSubmit,
    items,
    readOnly,
    readOnlyReason,
    initialValues,
    result,
    handleValuesChange,
    confirmLeave,
    handleSubmit,
    goToList,
    goToChains,
    goCreateItem,
  };
}
