import type { StatusTone } from '@shared/ui';

export interface ReplacementPillMeta {
  text: string;
  tone: StatusTone;
}

const ACTUALITY_DAYS = 14;

// all candidates in the sample have equal reliability, so only respondedAt tells them apart —
// and it's exchange_offers.updated_at, i.e. honestly it describes only how stale the refresh is
export function replacementPillMeta(
  respondedAt: string,
  now: Date = new Date(),
): ReplacementPillMeta {
  const responded = new Date(respondedAt).getTime();
  const withinActuality = responded >= now.getTime() - ACTUALITY_DAYS * 24 * 60 * 60 * 1000;
  return withinActuality
    ? { text: 'Актуальна', tone: 'success' }
    : { text: 'Давно не обновлялась', tone: 'warning' };
}
