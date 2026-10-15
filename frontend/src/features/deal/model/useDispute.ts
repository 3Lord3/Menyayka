import { useState } from 'react';

import { jsonStorage } from '@shared/lib/storage';

function disputeKey(chainId: number): string {
  return `deal:dispute:${chainId}`;
}

// there's no dispute endpoint on the backend — the flag is entirely client-side but must
// survive a reload
export function useDispute(chainId: number) {
  const [disputed, setDisputed] = useState<boolean>(
    () => jsonStorage.get<boolean>(disputeKey(chainId)) === true,
  );

  const openDispute = () => {
    jsonStorage.set(disputeKey(chainId), true);
    setDisputed(true);
  };

  return { disputed, openDispute };
}
