import { act, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { AxiosError } from 'axios';
import { useSearchParams } from 'react-router';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import {
  confirmChain,
  useChain,
  useReplacements,
  voteForRequest,
  withdrawVote,
  type Chain,
  type ReplacementOption,
} from '@entities/chain';

import { createTestQueryClient, renderWithProviders } from '@shared/testing/renderWithProviders';

import { ChainDetailPage } from './ChainDetailPage';

function queryOk(data: unknown) {
  return { data, isPending: false, isError: false, refetch: vi.fn() } as never;
}

function axiosError(status: number) {
  const error = new AxiosError('request failed');
  Object.assign(error, { response: { status } });
  return error;
}

vi.mock('@entities/chain', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@entities/chain')>();
  return {
    ...actual,
    useChain: vi.fn(),
    confirmChain: vi.fn(),
    voteForRequest: vi.fn(),
    withdrawVote: vi.fn(),
    useReplacements: vi.fn(),
  };
});

const mockedUseChain = vi.mocked(useChain);
const mockedConfirm = vi.mocked(confirmChain);
const mockedVote = vi.mocked(voteForRequest);
const mockedWithdraw = vi.mocked(withdrawVote);
const mockedUseReplacements = vi.mocked(useReplacements);

function mockReplacements(options: ReplacementOption[]) {
  mockedUseReplacements.mockReturnValue({
    data: options,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
  } as never);
}

function makeChain(overrides: Partial<Chain> = {}): Chain {
  return {
    id: 1,
    status: 'CANDIDATE',
    score: 0.9,
    length: 2,
    version: 1,
    currentRequestId: 101,
    currentPosition: 1,
    givesToPosition: 2,
    receivesFromPosition: 2,
    createdAt: '',
    updatedAt: '',
    participants: [
      {
        clusterId: 1,
        requestId: 101,
        position: 1,
        isCurrentUser: true,
        offeredItemId: 1,
        offeredItemTitle: 'Велосипед',
        offeredItemDescription: '',
        wantedDescription: 'Хочу фотоаппарат',
        requestStatus: 'ACTIVE' as const,
      },
      {
        clusterId: 2,
        requestId: 202,
        position: 2,
        isCurrentUser: false,
        offeredItemId: 2,
        offeredItemTitle: 'Зеркальный фотоаппарат Canon',
        offeredItemDescription: 'Полный комплект: камера, объектив, флешка и чехол',
        wantedDescription: 'Хочу велосипед',
        imageUrl: 'http://localhost:9000/photos/canon.jpg',
        requestStatus: 'ACTIVE' as const,
      },
    ],
    ...overrides,
  };
}

// the transition's target screen: shows which receive option reached the participants view
function OptionProbe() {
  const [searchParams] = useSearchParams();
  return <div>участники: {searchParams.get('option')}</div>;
}

describe('ChainDetailPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockReplacements([]);
  });

  afterEach(() => {
    // timer tests pin the date via vi.setSystemTime — restore the real Date.now()
    vi.useRealTimers();
  });

  it('shows the received item with its description', () => {
    mockedUseChain.mockReturnValue(queryOk(makeChain()));

    renderWithProviders(<ChainDetailPage />);

    expect(
      screen.getByRole('heading', { name: 'Зеркальный фотоаппарат Canon', level: 2 }),
    ).toBeInTheDocument();
    expect(screen.getByText('2 участника в цепочке')).toBeInTheDocument();
    expect(
      screen.getByText('Полный комплект: камера, объектив, флешка и чехол'),
    ).toBeInTheDocument();
  });

  it('opens the participants screen on button click', async () => {
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(queryOk(makeChain()));

    renderWithProviders(<ChainDetailPage />, {
      routes: [{ path: '/chains/1/participants', element: <div>участники</div> }],
    });

    await user.click(screen.getByRole('button', { name: 'Посмотреть всю цепочку' }));
    expect(await screen.findByText('участники')).toBeInTheDocument();
  });

  it('responds to the received candidate from the chain page', async () => {
    mockedVote.mockResolvedValue({
      chainId: 1,
      requestId: 101,
      targetRequestId: 202,
      vote: 'pending',
      votedAt: '2026-08-08T12:00:00Z',
      chainStatus: 'CANDIDATE',
    });
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(queryOk(makeChain()));

    renderWithProviders(<ChainDetailPage />);

    await user.click(screen.getByRole('button', { name: 'Откликнуться' }));
    await waitFor(() =>
      expect(mockedVote).toHaveBeenCalledWith(1, { requestId: 101, targetRequestId: 202 }),
    );
  });

  it('withdraws the pending vote from the chain page through the modal', async () => {
    mockedWithdraw.mockResolvedValue(undefined);
    const user = userEvent.setup();
    const chain = makeChain();
    chain.participants[1].vote = 'pending';
    mockedUseChain.mockReturnValue(queryOk(chain));

    renderWithProviders(<ChainDetailPage />);

    await user.click(screen.getByRole('button', { name: 'Отозвать отклик' }));
    await user.click(await screen.findByRole('button', { name: 'Да, отозвать' }));

    await waitFor(() =>
      expect(mockedWithdraw).toHaveBeenCalledWith(1, { requestId: 101, targetRequestId: 202 }),
    );
  });

  it('hides the vote button once the respond is approved or rejected', () => {
    const chain = makeChain();
    chain.participants[1].vote = 'approved';
    mockedUseChain.mockReturnValue(queryOk(chain));

    renderWithProviders(<ChainDetailPage />);

    expect(screen.queryByRole('button', { name: 'Откликнуться' })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Отозвать отклик' })).not.toBeInTheDocument();
  });

  // the respond button acts on the candidate with a pending response, otherwise on the first
  // without a response
  it('responds to the first candidate without a vote when the pool has several', async () => {
    const pool = Array.from({ length: 2 }, (_, index) => ({
      clusterId: 2,
      requestId: 202 + index,
      position: 2,
      isCurrentUser: false,
      offeredItemId: 20 + index,
      offeredItemTitle: `Фотоаппарат ${index + 1}`,
      offeredItemDescription: '',
      wantedDescription: 'Хочу велосипед',
      requestStatus: 'ACTIVE' as const,
    }));
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(
      queryOk(makeChain({ participants: [makeChain().participants[0], ...pool] })),
    );

    renderWithProviders(<ChainDetailPage />);

    await user.click(screen.getByRole('button', { name: 'Откликнуться' }));
    await waitFor(() =>
      expect(mockedVote).toHaveBeenCalledWith(1, { requestId: 101, targetRequestId: 202 }),
    );
  });

  it('shows the option from the link when the receiving pool has several', async () => {
    const pool = Array.from({ length: 2 }, (_, index) => ({
      clusterId: 2,
      requestId: 202 + index,
      position: 2,
      isCurrentUser: false,
      offeredItemId: 20 + index,
      offeredItemTitle: `Фотоаппарат ${index + 1}`,
      offeredItemDescription: '',
      wantedDescription: 'Хочу велосипед',
      requestStatus: 'ACTIVE' as const,
    }));
    mockedVote.mockResolvedValue({
      chainId: 1,
      requestId: 101,
      targetRequestId: 203,
      vote: 'pending',
      votedAt: '2026-08-08T12:00:00Z',
      chainStatus: 'CANDIDATE',
    });
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(
      queryOk(makeChain({ participants: [makeChain().participants[0], ...pool] })),
    );

    renderWithProviders(<ChainDetailPage />, { initialEntries: ['/chains/1?option=203'] });

    expect(screen.getByRole('heading', { name: 'Фотоаппарат 2', level: 2 })).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Откликнуться' }));
    await waitFor(() =>
      expect(mockedVote).toHaveBeenCalledWith(1, { requestId: 101, targetRequestId: 203 }),
    );
  });

  it('carries the selected option to the participants screen', async () => {
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(queryOk(makeChain()));

    renderWithProviders(<ChainDetailPage />, {
      initialEntries: ['/chains/1?option=202'],
      routes: [
        {
          path: '/chains/1/participants',
          element: <OptionProbe />,
        },
      ],
    });

    await user.click(screen.getByRole('button', { name: 'Посмотреть всю цепочку' }));
    expect(await screen.findByText('участники: 202')).toBeInTheDocument();
  });

  it('shows the probability badge once the chain is proposed', () => {
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'PROPOSED' })));

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('Высокая · 90%')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Требуются действия' })).toBeInTheDocument();
  });

  it('confirms participation from the chain item view', async () => {
    mockedConfirm.mockResolvedValue({ chainId: 1, status: 'FROZEN' });
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'PROPOSED' })));

    renderWithProviders(<ChainDetailPage />);

    await user.click(screen.getByRole('button', { name: 'Требуются действия' }));
    await user.click(await screen.findByRole('button', { name: 'Да' }));

    await waitFor(() => expect(mockedConfirm).toHaveBeenCalledWith(1));
  });

  it('shows the hard lock plaque and the shipment action on a frozen chain', () => {
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'FROZEN' })));

    renderWithProviders(<ChainDetailPage />);

    expect(
      screen.getByText('Товар жёстко заблокирован: изменить или удалить заявку нельзя'),
    ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Требуется действие' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Перейти к сделке' })).not.toBeInTheDocument();
  });

  it('shows the consent badge with the approved count on a proposed chain', () => {
    const chain = makeChain({ status: 'PROPOSED' });
    chain.participants[0].vote = 'pending';
    chain.participants[1].vote = 'approved';
    mockedUseChain.mockReturnValue(queryOk(chain));

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('1/2 согласий')).toBeInTheDocument();
  });

  it('shows M/M consents on a frozen chain', () => {
    const chain = makeChain({ status: 'FROZEN' });
    chain.participants[0].vote = 'approved';
    chain.participants[1].vote = 'approved';
    mockedUseChain.mockReturnValue(queryOk(chain));

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('2/2 согласий')).toBeInTheDocument();
  });

  // on FROZEN the same field carries the shipping deadline, so the row is gated by status
  it('shows the response deadline on a proposed chain', () => {
    vi.setSystemTime(new Date('2026-08-10T10:00:00Z'));
    mockedUseChain.mockReturnValue(
      queryOk(makeChain({ status: 'PROPOSED', freezeDeadlineAt: '2026-08-12T09:58:00Z' })),
    );

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('Осталось 47 ч 58 мин на ответ')).toBeInTheDocument();
  });

  it('hides the deadline row on a frozen chain even when the deadline is set', () => {
    vi.setSystemTime(new Date('2026-08-10T10:00:00Z'));
    mockedUseChain.mockReturnValue(
      queryOk(makeChain({ status: 'FROZEN', freezeDeadlineAt: '2026-08-12T09:58:00Z' })),
    );

    renderWithProviders(<ChainDetailPage />);

    expect(screen.queryByText(/Осталось .* на ответ/)).not.toBeInTheDocument();
  });

  // without a re-request at the deadline "Actions required" lives until the next poll
  it('refetches the chain right after the response deadline passes', () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-08-10T10:00:00Z'));
    mockedUseChain.mockReturnValue(
      queryOk(makeChain({ status: 'PROPOSED', freezeDeadlineAt: '2026-08-10T10:01:00Z' })),
    );
    const client = createTestQueryClient();
    const invalidate = vi.spyOn(client, 'invalidateQueries').mockResolvedValue(undefined);

    renderWithProviders(<ChainDetailPage />, { client });
    expect(invalidate).not.toHaveBeenCalled();

    act(() => {
      vi.advanceTimersByTime(62_000);
    });

    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['chains', 1] });
  });

  it('replaces the action with the confirmed line once my vote is approved', () => {
    const chain = makeChain({ status: 'PROPOSED' });
    chain.participants[0].vote = 'pending';
    chain.participants[1].vote = 'approved';
    mockedUseChain.mockReturnValue(queryOk(chain));

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('Вы подтвердили · ждём остальных')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Требуются действия' })).not.toBeInTheDocument();
  });

  // the candidate pool may be larger than the chain length — take the participant count from length
  it('counts participants by chain length, not by the pool size', () => {
    const pool = Array.from({ length: 5 }, (_, index) => ({
      clusterId: 2,
      requestId: 202 + index,
      position: 2,
      isCurrentUser: false,
      offeredItemId: 20 + index,
      offeredItemTitle: `Фотоаппарат ${index + 1}`,
      offeredItemDescription: '',
      wantedDescription: 'Хочу велосипед',
      requestStatus: 'ACTIVE' as const,
    }));
    mockedUseChain.mockReturnValue(
      queryOk(makeChain({ participants: [makeChain().participants[0], ...pool] })),
    );

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('2 участника в цепочке')).toBeInTheDocument();
    expect(
      screen.getByRole('heading', { name: 'Получаете: 5 вариантов', level: 2 }),
    ).toBeInTheDocument();
  });

  // a non-empty replacement pool is the only vacancy sign: a decline isn't visible in the chain body
  it('offers to pick a replacement when the pool is not empty', async () => {
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'PROPOSED' })));
    mockReplacements([
      {
        requestId: 42,
        offeredItemId: 17,
        title: 'Кофемашина капсульная',
        description: '',
        wantedDescription: 'Ищу фотоаппарат',
        reliability: 0.82,
        respondedAt: '2026-08-09T12:00:00Z',
      },
    ]);

    renderWithProviders(<ChainDetailPage />, {
      routes: [{ path: '/chains/1/replacement', element: <div>экран замены</div> }],
    });

    expect(
      screen.getByText('Участник отказался. Выберите замену, чтобы продолжить обмен'),
    ).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Требуются действия' })).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Требуется действие' }));
    expect(await screen.findByText('экран замены')).toBeInTheDocument();
  });

  // a disabled react-query keeps previous data: without a status check the pill would
  // survive the replacement confirmation
  it('drops the replacement plate once the chain leaves PROPOSED', () => {
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'FROZEN' })));
    mockReplacements([
      {
        requestId: 42,
        offeredItemId: 17,
        title: 'Кофемашина капсульная',
        description: '',
        wantedDescription: 'Ищу фотоаппарат',
        reliability: 0.82,
        respondedAt: '2026-08-09T12:00:00Z',
      },
    ]);

    renderWithProviders(<ChainDetailPage />);

    expect(
      screen.queryByText('Участник отказался. Выберите замену, чтобы продолжить обмен'),
    ).not.toBeInTheDocument();
  });

  it('leaves a healthy proposed chain without the replacement plate', () => {
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'PROPOSED' })));

    renderWithProviders(<ChainDetailPage />);

    expect(
      screen.queryByText('Участник отказался. Выберите замену, чтобы продолжить обмен'),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Требуются действия' })).toBeInTheDocument();
  });

  it('shows an error state with retry when the chain fails to load', async () => {
    const refetch = vi.fn();
    mockedUseChain.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
      refetch,
    } as never);

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('Что-то пошло не так')).toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: 'Повторить попытку' }));
    expect(refetch).toHaveBeenCalled();
  });

  it('shows the probability badge with the score on a candidate chain', () => {
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'CANDIDATE', score: 0.9 })));

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('Высокая · 90%')).toBeInTheDocument();
  });

  it('offers to decline from FROZEN through the secondary button', async () => {
    const user = userEvent.setup();
    mockedUseChain.mockReturnValue(queryOk(makeChain({ status: 'FROZEN' })));

    renderWithProviders(<ChainDetailPage />);

    await user.click(screen.getByRole('button', { name: 'Отказаться от сделки' }));
    expect(await screen.findByRole('button', { name: 'Да, отказаться' })).toBeInTheDocument();
  });

  it('shows the expired state when the detail request returns 410', () => {
    mockedUseChain.mockReturnValue({
      data: undefined,
      isPending: false,
      isError: true,
      error: axiosError(410),
    } as never);

    renderWithProviders(<ChainDetailPage />);

    expect(screen.getByText('Время истекло')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'К моим запросам' })).toBeInTheDocument();
  });
});
