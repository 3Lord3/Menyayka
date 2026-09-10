import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App as AntApp } from 'antd';

import { archiveItem } from '@entities/item';

import { getErrorMessage } from '@shared/lib/errorMessage';

export function useArchiveItem(onSuccess?: () => void) {
  const { message } = AntApp.useApp();
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (itemId: number) => archiveItem(itemId),
    onSuccess: () => {
      message.success('Товар удалён');
      queryClient.invalidateQueries({ queryKey: ['items'] });
      // the server deletes the item's related requests together with it — otherwise their
      // list stays stale in cache (staleTime 60s) and a click on a deleted request leads to 404
      queryClient.invalidateQueries({ queryKey: ['exchange-requests'] });
      onSuccess?.();
    },
    onError: (error) =>
      message.error(
        getErrorMessage(error, { 409: 'Товар уже участвует в сделке' }, 'Не удалось удалить товар'),
      ),
  });
}
