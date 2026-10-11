import { createContext, useContext } from 'react';

import type { RealtimeClient } from './client';

export const RealtimeContext = createContext<RealtimeClient | null>(null);

export function useRealtimeClient(): RealtimeClient | null {
  return useContext(RealtimeContext);
}
