import { describe, expect, it } from 'vitest';

import { replacementStage } from './replacementStage';

describe('replacementStage', () => {
  it('keeps the actor selecting until the invitation is sent', () => {
    expect(replacementStage('PROPOSED', false)).toBe('selecting');
  });

  it('waits for the candidate while the chain stays proposed', () => {
    expect(replacementStage('PROPOSED', true)).toBe('waiting');
  });

  // the candidate's confirmation is the last missing piece, so the chain goes straight to FROZEN
  it('reports success once the chain is frozen', () => {
    expect(replacementStage('FROZEN', true)).toBe('succeeded');
  });

  it('reports a rollback when the candidate declined or the chain fell apart', () => {
    expect(replacementStage('CANDIDATE', true)).toBe('rolledBack');
    expect(replacementStage('BROKEN', true)).toBe('rolledBack');
  });

  // outside PROPOSED the pool isn't requested, and an empty list would suggest dissolving a
  // live chain
  it('never offers the candidate list outside a proposed chain', () => {
    expect(replacementStage('IN_PROGRESS', false)).toBe('rolledBack');
    expect(replacementStage('COMPLETED', false)).toBe('rolledBack');
    expect(replacementStage(undefined, false)).toBe('rolledBack');
  });
});
