import { useSearchParams } from 'react-router';

// chain screens are one per chainId, but there are several variant cards for it: without
// binding to requestId the screen would expand the link back into the whole cluster of requests
export const RECEIVE_OPTION_PARAM = 'option';

export function useReceiveOption(): number | undefined {
  const [searchParams] = useSearchParams();
  const raw = searchParams.get(RECEIVE_OPTION_PARAM);
  if (!raw) return undefined;
  const requestId = Number(raw);
  return Number.isInteger(requestId) ? requestId : undefined;
}

export function receiveOptionQuery(requestId: number | undefined): string {
  return requestId === undefined ? '' : `?${RECEIVE_OPTION_PARAM}=${requestId}`;
}
