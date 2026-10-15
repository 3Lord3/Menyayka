import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App as AntApp } from 'antd';

import {
  confirmHandoff,
  confirmReceipt,
  invalidateChainQueries,
  sourceParticipant,
  type Chain,
  type FulfillmentResult,
} from '@entities/chain';

import { getErrorMessage } from '@shared/lib/errorMessage';

type DealFulfillmentKind = 'handoff' | 'receipt';

// The shipping/receiving endpoint error codes differ only by 409
const HANDOFF_ERROR_MESSAGES: Record<number, string> = {
  404: 'Цепочка не найдена',
  409: 'Цепочка ещё не готова к передаче товаров',
  422: 'Заявка не является закреплённым товаром этой цепочки',
};

const RECEIPT_ERROR_MESSAGES: Record<number, string> = {
  403: 'Только получатель товара может подтвердить получение',
  404: 'Цепочка не найдена',
  409: 'Передача товара ещё не подтверждена',
  422: 'Заявка не является закреплённым товаром этой цепочки',
};

// one mutation for both actions, differing only in requestId: for handoff — my request,
// for receipt — the source link's request whose item I'm picking up
export function useDealFulfillment(chain: Chain) {
  const { message } = AntApp.useApp();
  const queryClient = useQueryClient();

  const mutation = useMutation<FulfillmentResult, Error, DealFulfillmentKind>({
    mutationFn: (kind) => {
      if (kind === 'handoff') return confirmHandoff(chain.id, chain.currentRequestId);
      const source = sourceParticipant(chain);
      // shouldn't happen: the UI keeps the receiving button disabled while the source isn't
      // IN_PROGRESS
      if (!source) return Promise.reject(new Error('Источник товара не найден'));
      return confirmReceipt(chain.id, source.requestId);
    },
    onSuccess: (data, kind) => {
      invalidate();
      if (kind === 'receipt') {
        // the screen moves into the needed state itself — no separate success modal needed
        if (data.status === 'COMPLETED') message.success('Обмен завершён');
        return;
      }
      message.success('Отправка подтверждена');
    },
    onError: (error, kind) => {
      invalidate();
      message.error(
        getErrorMessage(
          error,
          kind === 'handoff' ? HANDOFF_ERROR_MESSAGES : RECEIPT_ERROR_MESSAGES,
          'Не удалось подтвердить операцию',
        ),
      );
    },
  });

  function invalidate() {
    invalidateChainQueries(queryClient);
  }

  function run(kind: DealFulfillmentKind) {
    // no confirmation modal: shipping is confirmed by the mandatory package photo,
    // and the endpoints are idempotent — a repeated click is gated by isPending
    mutation.mutateAsync(kind).then(
      () => undefined,
      () => undefined,
    );
  }

  return {
    confirmHandoff: () => run('handoff'),
    confirmReceipt: () => run('receipt'),
    isFulfilling: mutation.isPending,
  };
}
