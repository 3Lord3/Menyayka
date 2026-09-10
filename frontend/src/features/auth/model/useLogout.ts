import { useNavigate } from 'react-router';

import { useAppDispatch } from '@app/store/hooks';
import { logout } from '@app/store/slices/userSlice';

import { queryClient } from '@shared/api';

import { logoutRequest } from '../api/authApi';

export function useLogout() {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();

  return async function logoutAndRedirect() {
    try {
      // call happens before clearing localStorage — the interceptor still attaches the live token
      await logoutRequest();
    } catch {
      // JWT is stateless, the server doesn't invalidate the token — on a network failure
      // (or an expired token) local logout is still mandatory
    }
    dispatch(logout());
    // the query cache must not survive an account switch — otherwise the next user sees foreign items
    queryClient.clear();
    navigate('/login', { replace: true });
  };
}
