import { useEffect, useRef, useState } from 'react';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { App as AntApp } from 'antd';
import { isAxiosError } from 'axios';
import { useNavigate } from 'react-router';

import {
  declineChain,
  invalidateChainQueries,
  selectReplacement,
  useChain,
  useReplacements,
  type ReplacementOption,
} from '@entities/chain';

import { getErrorMessage } from '@shared/lib/errorMessage';

import { declineMessage } from './declineMessage';
import { EXPIRED_CHAIN_MESSAGE, isChainExpired } from './expiredChain';
import { replacementInvited } from './replacementInvited';
import { replacementStage, type ReplacementStage } from './replacementStage';

const WAITING_POLL_INTERVAL = 15_000;

export function useReplacementSelection(chainId?: number) {
  const navigate = useNavigate();
  const { message, modal } = AntApp.useApp();
  const queryClient = useQueryClient();
  // the record survives a reload: otherwise the waiting screen turns back into candidate selection
  const [invited, setInvited] = useState(() => replacementInvited.get(chainId));
  const [selectedId, setSelectedId] = useState<number | null>(null);
  // synchronous barrier against a repeated PUT: isPending becomes true only after a re-render
  const inviteInFlight = useRef(false);

  // the router reuses the element between /chains/1 and /chains/2/replacement, no unmount —
  // without re-sync a lazily-raised invitation would leak into a neighboring chain
  const [renderedChainId, setRenderedChainId] = useState(chainId);
  if (renderedChainId !== chainId) {
    setRenderedChainId(chainId);
    setInvited(replacementInvited.get(chainId));
    setSelectedId(null);
  }

  const {
    data: chain,
    isLoading: isChainLoading,
    isError: isChainError,
    error: chainError,
    refetch: refetchChain,
  } = useChain(chainId, {
    refetchInterval: (current) =>
      invited !== null && current?.status === 'PROPOSED' ? WAITING_POLL_INTERVAL : false,
  });

  // outside PROPOSED no pool exists and the query would return 409
  const {
    data: options = [],
    isLoading: isListLoading,
    isError: isListError,
    refetch: refetchOptions,
  } = useReplacements(chainId, {
    enabled: chain?.status === 'PROPOSED',
    refetchInterval:
      invited !== null && chain?.status === 'PROPOSED' ? WAITING_POLL_INTERVAL : false,
  });

  // the only thing bounding the record's lifetime: otherwise the next replacement on this
  // chain would start right at the waiting screen
  const chainStatus = chain?.status;
  useEffect(() => {
    if (chainStatus && chainStatus !== 'PROPOSED') {
      replacementInvited.clear(chainId);
    }
  }, [chainId, chainStatus]);

  // the invitee's decline doesn't change the chain status, so a reopened vacancy is seen
  // only via a non-empty pool; the invited one is already gone from the fresh pool, and its
  // presence is read as data from the previous choice — otherwise a race with invalidation
  // after PUT would reset the stage
  const invitedRequestId = invited?.requestId ?? null;
  const vacancyReopened =
    invited !== null &&
    chain?.status === 'PROPOSED' &&
    options.length > 0 &&
    !options.some((option) => option.requestId === invitedRequestId);
  useEffect(() => {
    if (vacancyReopened) replacementInvited.clear(chainId);
  }, [chainId, vacancyReopened]);

  const isChainNotFound = isAxiosError(chainError) && chainError.response?.status === 404;
  const stage: ReplacementStage = isChainNotFound
    ? 'rolledBack'
    : replacementStage(chain?.status, invited !== null && !vacancyReopened);
  const isLoading = isListLoading || isChainLoading;
  // a 404 chain is not a load error but a normal rollback; a pool error crashes the screen
  // only where the pool is needed, otherwise its 403/409 on a closed vacancy would replace the
  // waiting screen
  const isError = (isChainError && !isChainNotFound) || (stage === 'selecting' && isListError);

  function refetch() {
    void refetchChain();
    void refetchOptions();
  }

  // after a background refetch the selected request may disappear — the invite button turns
  // off by itself
  const selectedOption = options.find((option) => option.requestId === selectedId) ?? null;

  const inviteMutation = useMutation({
    mutationFn: (option: ReplacementOption) =>
      selectReplacement(chainId as number, option.requestId),
    onSuccess: (_result, option) => {
      setInvited({ requestId: option.requestId, option });
      replacementInvited.set(chainId, option);
      // both the chain card and the replacement pool sit under ['chains'] — one invalidation
      // covers both
      queryClient.invalidateQueries({ queryKey: ['chains'] });
      // its own key with a 60s staleTime: otherwise "Exchange options" shows the old composition
      // for another minute
      queryClient.invalidateQueries({ queryKey: ['exchange-options'] });
    },
    onError: (error) => {
      if (isChainExpired(error)) {
        message.warning(EXPIRED_CHAIN_MESSAGE);
        invalidateChainQueries(queryClient);
        navigate('/exchange-requests');
        return;
      }
      message.error(
        getErrorMessage(
          error,
          {
            403: 'Замену уже выбрали',
            404: 'Цепочка не найдена',
            409: 'Цепочка изменилась: обновите варианты',
            422: 'Этот вариант больше не доступен',
          },
          'Не удалось пригласить замену',
        ),
      );
      if (isAxiosError(error) && error.response) {
        const status = error.response.status;
        // the actor changed or the vacancy closed — nothing to repeat
        if (status === 403) {
          navigate(`/chains/${chainId}`);
          return;
        }
        if (status === 404) {
          navigate('/exchange-requests');
          return;
        }
        if (status === 409) {
          queryClient.invalidateQueries({ queryKey: ['chains'] });
          return;
        }
      }
      refetchOptions();
    },
  });

  function invite() {
    if (inviteInFlight.current || selectedOption === null) return;
    inviteInFlight.current = true;
    inviteMutation.mutate(selectedOption, {
      onSettled: () => {
        inviteInFlight.current = false;
      },
    });
  }

  const abandonMutation = useMutation({
    mutationFn: () => declineChain(chainId as number),
    onSuccess: (result) => {
      // the chain no longer exists for us — nowhere left to see it in another status and
      // clear the record
      replacementInvited.clear(chainId);
      queryClient.invalidateQueries({ queryKey: ['chains'] });
      // a dissolved chain must drop from the list right away, not linger for a minute (staleTime)
      queryClient.invalidateQueries({ queryKey: ['exchange-options'] });
      // the server may roll the chain back to CANDIDATE or keep it PROPOSED under replacement
      // pickup — promising "dissolved" in advance is wrong, the outcome comes from the response
      message.success(
        result.status === 'BROKEN' ? 'Цепочка расформирована' : declineMessage(result.status),
      );
      // we exit the chain on any outcome, so /chains/{id} is closed for us
      const requestId = chain?.currentRequestId;
      navigate(requestId ? `/exchange-requests/${requestId}` : '/exchange-requests');
    },
    onError: (error) => {
      if (isChainExpired(error)) {
        message.warning(EXPIRED_CHAIN_MESSAGE);
        invalidateChainQueries(queryClient);
        navigate('/exchange-requests');
        return;
      }
      message.error(
        getErrorMessage(
          error,
          { 403: 'Это не ваша цепочка', 404: 'Цепочка не найдена' },
          'Не удалось расформировать цепочку',
        ),
      );
    },
  });

  function abandon() {
    modal.confirm({
      title: 'Отказаться от замены?',
      content: 'Вы выйдете из цепочки: если заменить вас будет некем, она расформируется',
      okText: 'Да, отказаться',
      okButtonProps: { danger: true },
      cancelText: 'Нет',
      onOk: () => abandonMutation.mutate(),
    });
  }

  return {
    options,
    isLoading,
    isError,
    refetch,
    selectedId: selectedOption?.requestId ?? null,
    setSelectedId,
    invite,
    isInviting: inviteMutation.isPending,
    abandon,
    isAbandoning: abandonMutation.isPending,
    stage,
    chain,
    invitedOption: vacancyReopened ? null : (invited?.option ?? null),
  };
}
