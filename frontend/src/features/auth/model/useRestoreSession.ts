import { useEffect } from 'react';

import { useAppDispatch } from '@app/store/hooks';
import { sessionRestored } from '@app/store/slices/userSlice';

import { tokenStorage } from '@shared/lib/storage';

import { fetchMeRequest } from '../api/authApi';

// the localStorage token has no fresh profile data, so we pull it from the server;
// an expired token is already handled by the apiClient interceptor — no 401 handling here
export function useRestoreSession() {
  const dispatch = useAppDispatch();

  useEffect(() => {
    if (!tokenStorage.get()) return;

    let cancelled = false;

    fetchMeRequest()
      .then((user) => {
        if (!cancelled) dispatch(sessionRestored(user));
      })
      .catch(() => {
        // a network error or 401 (handled by the interceptor) — leave the session alone
      });

    return () => {
      cancelled = true;
    };
  }, [dispatch]);
}
