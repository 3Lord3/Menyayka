import type { ChainStatus } from '@entities/chain';

export type ReplacementStage = 'selecting' | 'waiting' | 'succeeded' | 'rolledBack';

export function replacementStage(
  status: ChainStatus | undefined,
  invited: boolean,
): ReplacementStage {
  if (status === 'PROPOSED') return invited ? 'waiting' : 'selecting';
  if (status === 'FROZEN') return 'succeeded';
  // fallback goes here, not to 'selecting': outside PROPOSED the pool isn't requested, and an
  // empty list with a "Dissolve chain" button would suggest tearing down a live exchange
  return 'rolledBack';
}
