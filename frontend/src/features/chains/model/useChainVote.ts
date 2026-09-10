import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App as AntApp } from 'antd';

import {
  invalidateChainQueries,
  voteForRequest,
  withdrawVote,
  type ChainVoteResult,
} from '@entities/chain';

import { getErrorMessage } from '@shared/lib/errorMessage';

export interface VoteTarget {
  chainId: number;
  requestId: number;
  targetRequestId: number;
}

// a closed response ring moves the chain to PROPOSED on the backend itself — without reading
// the status from the response and invalidating, the screen would keep stale actions and catch a 409
export function useChainVote(onConflict?: () => void) {
  const { message, modal } = AntApp.useApp();
  const queryClient = useQueryClient();

  const mutation = useMutation<ChainVoteResult | void, Error, VoteTarget & { active: boolean }>({
    mutationFn: ({
      chainId,
      requestId,
      targetRequestId,
      active,
    }: VoteTarget & { active: boolean }) =>
      active
        ? voteForRequest(chainId, { requestId, targetRequestId })
        : withdrawVote(chainId, { requestId, targetRequestId }),
    onSuccess: (data, { active }) => {
      if (!active) {
        message.success('Отклик отозван');
      } else if (data?.chainStatus === 'PROPOSED') {
        message.success('Цепочка собрана');
      } else {
        message.success('Отклик отправлен');
      }
      // the chain status may be stale (someone else may have closed the ring) — reload the data
      invalidateChainQueries(queryClient);
    },
    onError: (error) => {
      invalidateChainQueries(queryClient);
      message.error(
        getErrorMessage(
          error,
          {
            403: 'Это не ваша заявка',
            404: 'Цепочка не найдена',
            409: 'Цепочка изменилась: обновите варианты',
            422: 'Некорректный вариант обмена',
          },
          'Не удалось отправить отклик',
        ),
      );
      onConflict?.();
    },
  });

  // a response is reversible, so confirmation is needed only to withdraw it
  function confirmVote(target: VoteTarget, active: boolean) {
    if (active) {
      mutation.mutate({ ...target, active: true });
      return;
    }
    modal.confirm({
      title: 'Отозвать отклик?',
      content: 'Отклик будет снят, цепочка вернётся в список вариантов.',
      okText: 'Да, отозвать',
      okButtonProps: { danger: true },
      cancelText: 'Нет',
      centered: true,
      onOk: () => mutation.mutate({ ...target, active: false }),
    });
  }

  return { confirmVote, isVoting: mutation.isPending };
}
