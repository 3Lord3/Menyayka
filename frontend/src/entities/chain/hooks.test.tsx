import { renderHook } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { chainQueryOptions, useChain, useChains, useExchangeOptions } from './hooks';

const mocks = vi.hoisted(() => ({
  useQuery: vi.fn(),
  useQueries: vi.fn(),
  realtimeConnected: false,
}));

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>();
  return { ...actual, useQuery: mocks.useQuery, useQueries: mocks.useQueries };
});

// the whole point of the layer is the live-socket gate, so it is driven directly
vi.mock('@shared/realtime', () => ({
  useRealtimeConnected: () => mocks.realtimeConnected,
}));

vi.mock('./api', () => ({
  fetchChain: vi.fn(),
  fetchExchangeOptions: vi.fn(),
  fetchReplacements: vi.fn(),
}));

const PROPOSED = { state: { data: { status: 'PROPOSED' } } };

describe('chainQueryOptions', () => {
  it('polls fast while there is no live socket', () => {
    expect(chainQueryOptions(1, false).refetchInterval(PROPOSED)).toBe(30_000);
  });

  it('stretches to the slow safety net while the socket is live', () => {
    expect(chainQueryOptions(1, true).refetchInterval(PROPOSED)).toBe(5 * 60_000);
  });

  it('does not poll a chain in a settled status', () => {
    const settled = { state: { data: { status: 'COMPLETED' } } };
    expect(chainQueryOptions(1, false).refetchInterval(settled)).toBe(false);
    expect(chainQueryOptions(1, true).refetchInterval(settled)).toBe(false);
  });
});

describe('realtime-aware polling wiring', () => {
  beforeEach(() => {
    mocks.useQuery.mockReset();
    mocks.useQueries.mockReset();
    mocks.realtimeConnected = false;
  });

  it('passes the live-socket state into useChain', () => {
    mocks.realtimeConnected = true;
    renderHook(() => useChain(1));

    expect(mocks.useQuery.mock.calls[0][0].refetchInterval(PROPOSED)).toBe(5 * 60_000);
  });

  it('applies the interval to every query of useChains', () => {
    renderHook(() => useChains([1, 2]));

    const { queries } = mocks.useQueries.mock.calls[0][0];
    expect(
      queries.map((query: { refetchInterval: (q: unknown) => unknown }) =>
        query.refetchInterval(PROPOSED),
      ),
    ).toEqual([30_000, 30_000]);
  });

  it('keys useExchangeOptions off the live socket and a pending option', () => {
    mocks.realtimeConnected = true;
    renderHook(() => useExchangeOptions(5));

    const options = mocks.useQuery.mock.calls[0][0];
    expect(options.refetchInterval({ state: { data: [{ status: 'PROPOSED' }] } })).toBe(5 * 60_000);
    expect(options.refetchInterval({ state: { data: [{ status: 'ACCEPTED' }] } })).toBe(false);
  });
});
