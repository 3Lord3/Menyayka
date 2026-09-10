import type { StatusTone } from '@shared/ui';

export type RequestStatus = 'ACTIVE' | 'IN_PROPOSAL' | 'LOCKED' | 'DONE' | 'IN_PROGRESS';

// The backend puts the given item's name only in the request list
// (GET /exchange-offers); in details (GET /exchange-offers/:id) it's absent.
export interface ExchangeRequest {
  id: number;
  offeredItemId: number;
  offeredItemTitle?: string;
  wantedDescription: string;
  // the wanted category; doesn't take part in matching (the backend builds the embedding
  // only from wantedDescription) — a display-only field
  wantedCategory: string;
  status: RequestStatus;
  version: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateRequestPayload {
  offeredItemId: number;
  wantedDescription: string;
  wantedCategory?: string;
}

// the number of found chains the backend doesn't return in the create response — the frontend
// takes it from exchange-options
export interface CreateRequestResult {
  request: ExchangeRequest;
  matching: { createdCandidateChains: number };
}

export interface UpdateRequestPayload {
  offeredItemId: number;
  wantedDescription: string;
  wantedCategory?: string;
  version: number;
}

export const REQUEST_STATUS_META: Record<RequestStatus, { label: string; tone: StatusTone }> = {
  ACTIVE: { label: 'Активен', tone: 'success' },
  IN_PROPOSAL: { label: 'В процессе', tone: 'warning' },
  LOCKED: { label: 'Зарезервирован', tone: 'neutral' },
  DONE: { label: 'Завершён', tone: 'neutral' },
  IN_PROGRESS: { label: 'Выполняется', tone: 'neutral' },
};

// REMOVED the backend filters out from the list and details, so such a status doesn't exist in the UI

export function isRequestEditable(status: RequestStatus): boolean {
  return status === 'ACTIVE' || status === 'IN_PROPOSAL';
}
