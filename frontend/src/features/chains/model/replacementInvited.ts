import type { ReplacementOption } from '@entities/chain';

const KEY_PREFIX = 'menyayka_replacement_invited';

interface InvitedRecord {
  // null — a previous-version record that stored only the fact of the invitation
  requestId: number | null;
  option: ReplacementOption | null;
}

function key(chainId: number): string {
  return `${KEY_PREFIX}_${chainId}`;
}

// The invitation isn't visible in the chain body, and PUT /replacement isn't idempotent:
// without a record a reload returns the screen to candidate selection and, with an empty pool,
// offers to dissolve a healthy chain. localStorage, not sessionStorage — a new tab would show
// the same thing.
export const replacementInvited = {
  get(chainId?: number): InvitedRecord | null {
    if (!chainId) return null;
    const raw = localStorage.getItem(key(chainId));
    if (!raw) return null;
    try {
      const parsed: unknown = JSON.parse(raw);
      if (parsed && typeof parsed === 'object' && 'requestId' in parsed) {
        const record = parsed as InvitedRecord;
        if (typeof record.requestId === 'number') {
          return { requestId: record.requestId, option: record.option ?? null };
        }
      }
    } catch {
      // a previous-version record ('1') — there was an invitation, but the candidate is unknown
    }
    return { requestId: null, option: null };
  },
  set(chainId: number | undefined, option: ReplacementOption): void {
    if (!chainId) return;
    const record: InvitedRecord = { requestId: option.requestId, option };
    localStorage.setItem(key(chainId), JSON.stringify(record));
  },
  clear(chainId?: number): void {
    if (!chainId) return;
    localStorage.removeItem(key(chainId));
  },
};
