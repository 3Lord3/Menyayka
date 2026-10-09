import { useRestoreSession } from '@features/auth';

import { RealtimeProvider } from '@shared/realtime';

import { AppProviders } from './providers';
import { RouterProvider } from './router/RouterProvider';

// not in AppLayout: the password-change screen sits outside the layout and remounts it on
// navigation, and session restore must happen exactly once per start
function SessionRestorer() {
  useRestoreSession();
  return null;
}

export function App() {
  return (
    <AppProviders>
      <RealtimeProvider>
        <SessionRestorer />
        <RouterProvider />
      </RealtimeProvider>
    </AppProviders>
  );
}
