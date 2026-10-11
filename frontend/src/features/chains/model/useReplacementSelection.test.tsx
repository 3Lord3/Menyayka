import type { ReactNode } from 'react';

import { QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, screen, waitFor } from '@testing-library/react';
import { App as AntApp } from 'antd';
import { AxiosError } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  declineChain,
  selectReplacement,
  useChain,
  useReplacements,
  type Chain,
  type ReplacementOption,
} from '@entities/chain';

import { createTestQueryClient } from '@shared/testing/renderWithProviders';

import { replacementInvited } from './replacementInvited';
import { useReplacementSelection } from './useReplacementSelection';

const navigate = vi.fn();

const realtime = vi.hoisted(() => ({ connected: false }));
vi.mock('@shared/realtime', () => ({ useRealtimeConnected: () => realtime.connected }));

vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return { ...actual, useNavigate: () => navigate };
});

vi.mock('@entities/chain', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@entities/chain')>();
  return {
    ...actual,
    useChain: vi.fn(),
    useReplacements: vi.fn(),
    selectReplacement: vi.fn(),
    declineChain: vi.fn(),
  };
});

const mockedUseChain = vi.mocked(useChain);
const mockedUseReplacements = vi.mocked(useReplacements);
const mockedSelect = vi.mocked(selectReplacement);
const mockedDecline = vi.mocked(declineChain);

const OPTION: ReplacementOption = {
  requestId: 42,
  offeredItemId: 17,
  title: 'Кофемашина капсульная',
  description: 'Почти не использовалась',
  wantedDescription: 'Ищу фотоаппарат',
  reliability: 0.82,
  respondedAt: '2026-08-09T12:00:00Z',
};

function makeChain(overrides: Partial<Chain> = {}): Chain {
  return {
    id: 1,
    status: 'PROPOSED',
    score: 0.9,
    length: 2,
    version: 1,
    currentRequestId: 101,
    currentPosition: 0,
    givesToPosition: 1,
    receivesFromPosition: 1,
    createdAt: '',
    updatedAt: '',
    participants: [],
    ...overrides,
  };
}

const refetchOptions = vi.fn();
const refetchChain = vi.fn();

function mockChainQuery(chain: Chain | undefined, extra: Record<string, unknown> = {}) {
  mockedUseChain.mockReturnValue({
    data: chain,
    isLoading: false,
    isError: false,
    error: null,
    refetch: refetchChain,
    ...extra,
  } as never);
}

function axiosError(status: number) {
  const error = new AxiosError('request failed');
  Object.assign(error, { response: { status } });
  return error;
}

// the realtime gate is asserted through the options the hook hands to the entity queries
type ChainPollOptions = { refetchInterval: (chain: { status?: string }) => number | false };
type PoolPollOptions = { refetchInterval?: number | false };

let queryClient = createTestQueryClient();

function wrapper({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <AntApp>{children}</AntApp>
    </QueryClientProvider>
  );
}

// candidate selection is separate from the invitation: invite() reads the chosen variant
// from the already-rendered pool
async function renderSelecting(options: ReplacementOption[] = [OPTION]) {
  mockedUseReplacements.mockReturnValue({
    data: options,
    isLoading: false,
    isError: false,
    refetch: refetchOptions,
  } as never);
  const view = renderHook(() => useReplacementSelection(1), { wrapper });
  act(() => view.result.current.setSelectedId(OPTION.requestId));
  return view;
}

describe('useReplacementSelection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // the "invitation sent" record lives in localStorage — otherwise tests leak into each other
    localStorage.clear();
    queryClient = createTestQueryClient();
    mockChainQuery(makeChain());
  });

  it('moves to waiting and refreshes the chain cache after a successful invite', async () => {
    mockedSelect.mockResolvedValue({ chainId: 1, requestId: 42, status: 'PROPOSED' });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = await renderSelecting();

    await act(async () => {
      result.current.invite();
    });

    expect(mockedSelect).toHaveBeenCalledWith(1, 42);
    await waitFor(() => expect(result.current.stage).toBe('waiting'));
    expect(invalidate.mock.calls.map(([options]) => options?.queryKey)).toContainEqual(['chains']);
  });

  it('re-fetches the pool and stays selecting when the candidate is gone', async () => {
    mockedSelect.mockRejectedValue(axiosError(422));
    const { result } = await renderSelecting();

    await act(async () => {
      result.current.invite();
    });

    expect(await screen.findByText('Этот вариант больше не доступен')).toBeInTheDocument();
    await waitFor(() => expect(refetchOptions).toHaveBeenCalled());
    expect(result.current.stage).toBe('selecting');
  });

  it('sends the actor back to the chain when the vacancy is already filled', async () => {
    mockedSelect.mockRejectedValue(axiosError(403));
    const { result } = await renderSelecting();

    await act(async () => {
      result.current.invite();
    });

    expect(await screen.findByText('Замену уже выбрали')).toBeInTheDocument();
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/chains/1'));
  });

  // the quick-replacement deadline passed — nothing to invite, the chain went to rollback
  it('warns and leaves for the requests when the replacement deadline expires on invite', async () => {
    mockedSelect.mockRejectedValue(axiosError(410));
    const { result } = await renderSelecting();

    await act(async () => {
      result.current.invite();
    });

    expect(await screen.findByText('Время истекло, цепочка распалась')).toBeInTheDocument();
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/exchange-requests'));
  });

  // the server isn't idempotent: a repeated PUT with the same requestId would return 422
  it('sends a single request when invite is called twice in a row', async () => {
    mockedSelect.mockImplementation(() => new Promise(() => {}));
    const { result } = await renderSelecting();

    await act(async () => {
      result.current.invite();
      result.current.invite();
    });

    expect(mockedSelect).toHaveBeenCalledTimes(1);
  });

  it('does nothing until a candidate is selected', async () => {
    mockedUseReplacements.mockReturnValue({
      data: [OPTION],
      isLoading: false,
      isError: false,
      refetch: refetchOptions,
    } as never);
    const { result } = renderHook(() => useReplacementSelection(1), { wrapper });

    await act(async () => {
      result.current.invite();
    });

    expect(mockedSelect).not.toHaveBeenCalled();
  });

  it('confirms abandoning and leaves for the request options', async () => {
    mockedDecline.mockResolvedValue({ chainId: 1, status: 'BROKEN', replacementAvailable: false });
    const { result } = await renderSelecting();

    act(() => result.current.abandon());
    const confirm = await screen.findByRole('button', { name: 'Да, отказаться' });
    await act(async () => {
      confirm.click();
    });

    await waitFor(() => expect(mockedDecline).toHaveBeenCalledWith(1));
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/exchange-requests/101'));
  });

  it('treats a missing chain as a rollback rather than a load failure', async () => {
    mockChainQuery(undefined, { isError: true, error: axiosError(404) });
    const { result } = await renderSelecting();

    expect(result.current.stage).toBe('rolledBack');
    expect(result.current.isError).toBe(false);
  });

  it('reports a real chain load failure', async () => {
    mockChainQuery(undefined, { isError: true, error: axiosError(500) });
    const { result } = await renderSelecting();

    expect(result.current.isError).toBe(true);
  });

  // the pool stays enabled during the whole waiting (the chain is still PROPOSED); its
  // 403/409 on a closed vacancy must not replace "Waiting for candidate's answer" with a
  // generic error
  it('keeps the waiting screen when the stale pool query fails', async () => {
    mockedSelect.mockResolvedValue({ chainId: 1, requestId: 42, status: 'PROPOSED' });
    mockedUseReplacements.mockReturnValue({
      data: [OPTION],
      isLoading: false,
      isError: true,
      refetch: refetchOptions,
    } as never);
    const { result } = renderHook(() => useReplacementSelection(1), { wrapper });
    act(() => result.current.setSelectedId(OPTION.requestId));

    expect(result.current.isError).toBe(true);

    await act(async () => {
      result.current.invite();
    });

    await waitFor(() => expect(result.current.stage).toBe('waiting'));
    expect(result.current.isError).toBe(false);
  });

  it('drops a selection that no longer exists in the refreshed pool', async () => {
    const { result, rerender } = await renderSelecting();
    expect(result.current.selectedId).toBe(OPTION.requestId);

    mockedUseReplacements.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
      refetch: refetchOptions,
    } as never);
    rerender();

    expect(result.current.selectedId).toBeNull();
    await act(async () => {
      result.current.invite();
    });
    expect(mockedSelect).not.toHaveBeenCalled();
  });

  it('refreshes the chain itself, not just the pool, on retry', async () => {
    const { result } = await renderSelecting();

    act(() => result.current.refetch());

    expect(refetchChain).toHaveBeenCalled();
  });

  // PUT isn't idempotent and the in-body vacancy isn't visible: without a reload-surviving
  // record the screen would return to 'selecting' and offer to dissolve a healthy chain
  it('restores the waiting screen after a reload', async () => {
    mockedSelect.mockResolvedValue({ chainId: 1, requestId: 42, status: 'PROPOSED' });
    const first = await renderSelecting();

    await act(async () => {
      first.result.current.invite();
    });
    await waitFor(() => expect(first.result.current.stage).toBe('waiting'));
    first.unmount();

    // reload: a new hook instance, in-memory state is lost, the chain is still PROPOSED
    const { result } = renderHook(() => useReplacementSelection(1), { wrapper });

    expect(result.current.stage).toBe('waiting');
  });

  // the invitee's decline doesn't change the chain's status — the reopened vacancy is visible
  // only through the pool
  it('returns to selecting when the invited candidate declined and the vacancy reopened', async () => {
    mockedSelect.mockResolvedValue({ chainId: 1, requestId: 42, status: 'PROPOSED' });
    const { result, rerender } = await renderSelecting();

    await act(async () => {
      result.current.invite();
    });
    await waitFor(() => expect(result.current.stage).toBe('waiting'));

    const other: ReplacementOption = { ...OPTION, requestId: 99, title: 'Фотоаппарат плёночный' };
    mockedUseReplacements.mockReturnValue({
      data: [other],
      isLoading: false,
      isError: false,
      refetch: refetchOptions,
    } as never);
    rerender();

    expect(result.current.stage).toBe('selecting');
    await waitFor(() => expect(replacementInvited.get(1)).toBeNull());
  });

  // right after PUT the invalidation hasn't arrived yet and the pool is unchanged — the
  // invited one is still in it. Reading that as a new vacancy would throw the actor back to
  // candidate selection.
  it('keeps waiting while the pool still holds the invited candidate', async () => {
    mockedSelect.mockResolvedValue({ chainId: 1, requestId: 42, status: 'PROPOSED' });
    const { result, rerender } = await renderSelecting();

    await act(async () => {
      result.current.invite();
    });
    await waitFor(() => expect(result.current.stage).toBe('waiting'));

    rerender();

    expect(result.current.stage).toBe('waiting');
    expect(replacementInvited.get(1)).not.toBeNull();
  });

  it('restores the invited candidate card after a reload', async () => {
    mockedSelect.mockResolvedValue({ chainId: 1, requestId: 42, status: 'PROPOSED' });
    const first = await renderSelecting();

    await act(async () => {
      first.result.current.invite();
    });
    await waitFor(() => expect(first.result.current.invitedOption).toEqual(OPTION));
    first.unmount();

    mockedUseReplacements.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
      refetch: refetchOptions,
    } as never);
    const { result } = renderHook(() => useReplacementSelection(1), { wrapper });

    expect(result.current.stage).toBe('waiting');
    expect(result.current.invitedOption).toEqual(OPTION);
  });

  it('forgets the invitation once the chain leaves PROPOSED', async () => {
    replacementInvited.set(1, OPTION);
    // the candidate declined — the chain returned to CANDIDATE, the record is stale
    mockChainQuery(makeChain({ status: 'CANDIDATE' }));
    const { unmount } = await renderSelecting();
    await waitFor(() => expect(replacementInvited.get(1)).toBeNull());
    unmount();

    mockChainQuery(makeChain({ status: 'PROPOSED' }));
    const { result } = renderHook(() => useReplacementSelection(1), { wrapper });

    expect(result.current.stage).toBe('selecting');
  });

  // a decline doesn't always dissolve the chain: the server may roll it back to CANDIDATE,
  // and promising "Chain dissolved" then is wrong — the outcome is visible only from the response
  it('reports the real outcome when the chain survives the refusal', async () => {
    mockedDecline.mockResolvedValue({
      chainId: 1,
      status: 'CANDIDATE',
      replacementAvailable: true,
    });
    const { result } = await renderSelecting();

    act(() => result.current.abandon());
    const confirm = await screen.findByRole('button', { name: 'Да, отказаться' });
    await act(async () => {
      confirm.click();
    });

    expect(
      await screen.findByText('Вы вышли из сделки. Цепочка вернулась к сбору откликов'),
    ).toBeInTheDocument();
    expect(screen.queryByText('Цепочка расформирована')).not.toBeInTheDocument();
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/exchange-requests/101'));
  });

  it('warns and leaves for the requests when the replacement deadline expires on abandon', async () => {
    mockedDecline.mockRejectedValue(axiosError(410));
    const { result } = await renderSelecting();

    act(() => result.current.abandon());
    const confirm = await screen.findByRole('button', { name: 'Да, отказаться' });
    await act(async () => {
      confirm.click();
    });

    expect(await screen.findByText('Время истекло, цепочка распалась')).toBeInTheDocument();
    await waitFor(() => expect(navigate).toHaveBeenCalledWith('/exchange-requests'));
  });

  // same route, different param — the router doesn't unmount the element, and without
  // re-sync the first chain's invitation would open the second right on "Waiting for answer"
  it('forgets the previous chain state when the chain id changes', async () => {
    replacementInvited.set(1, OPTION);
    mockedUseReplacements.mockReturnValue({
      data: [OPTION],
      isLoading: false,
      isError: false,
      refetch: refetchOptions,
    } as never);
    const { result, rerender } = renderHook((chainId: number) => useReplacementSelection(chainId), {
      wrapper,
      initialProps: 1,
    });
    act(() => result.current.setSelectedId(OPTION.requestId));
    expect(result.current.stage).toBe('waiting');

    rerender(2);

    expect(result.current.stage).toBe('selecting');
    expect(result.current.invitedOption).toBeNull();
    expect(result.current.selectedId).toBeNull();
  });

  it('invalidates the request options after disbanding and after inviting', async () => {
    mockedDecline.mockResolvedValue({ chainId: 1, status: 'BROKEN', replacementAvailable: false });
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    const { result } = await renderSelecting();

    act(() => result.current.abandon());
    const confirm = await screen.findByRole('button', { name: 'Да, отказаться' });
    await act(async () => {
      confirm.click();
    });

    await waitFor(() =>
      expect(invalidate.mock.calls.map(([options]) => options?.queryKey)).toContainEqual([
        'exchange-options',
      ]),
    );
  });

  // while the socket is live the wait resolves by events; polling is a slow safety net
  it('slows the waiting poll while the socket is live', () => {
    realtime.connected = true;
    try {
      replacementInvited.set(1, OPTION);
      mockedUseReplacements.mockReturnValue({
        data: [OPTION],
        isLoading: false,
        isError: false,
        refetch: refetchOptions,
      } as never);

      renderHook(() => useReplacementSelection(1), { wrapper });

      const chainOptions = mockedUseChain.mock.calls.at(-1)?.[1] as ChainPollOptions;
      expect(chainOptions.refetchInterval({ status: 'PROPOSED' })).toBe(5 * 60_000);
      const poolOptions = mockedUseReplacements.mock.calls.at(-1)?.[1] as PoolPollOptions;
      expect(poolOptions.refetchInterval).toBe(5 * 60_000);
    } finally {
      realtime.connected = false;
    }
  });

  // a rejected handshake or a dropped socket must not stretch the refresh window
  it('keeps the fast waiting poll while the socket is down', () => {
    replacementInvited.set(1, OPTION);
    mockedUseReplacements.mockReturnValue({
      data: [OPTION],
      isLoading: false,
      isError: false,
      refetch: refetchOptions,
    } as never);

    renderHook(() => useReplacementSelection(1), { wrapper });

    const chainOptions = mockedUseChain.mock.calls.at(-1)?.[1] as ChainPollOptions;
    expect(chainOptions.refetchInterval({ status: 'PROPOSED' })).toBe(15_000);
    const poolOptions = mockedUseReplacements.mock.calls.at(-1)?.[1] as PoolPollOptions;
    expect(poolOptions.refetchInterval).toBe(15_000);
  });
});
