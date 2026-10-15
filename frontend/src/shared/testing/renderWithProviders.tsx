import type { ReactNode } from 'react';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, type RenderResult } from '@testing-library/react';
import { App as AntApp } from 'antd';
import { MemoryRouter, Route, Routes } from 'react-router';

export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
}

interface RenderWithProvidersOptions {
  initialEntries?: string[];
  routes?: { path: string; element: ReactNode }[];
  // its own client is needed by tests that watch cache invalidation
  client?: QueryClient;
}

// ui renders as the catch-all route, while routes — as concrete target screens to test navigation
export function renderWithProviders(
  ui: ReactNode,
  { initialEntries = ['/'], routes = [], client }: RenderWithProvidersOptions = {},
): RenderResult {
  const queryClient = client ?? createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <AntApp>
        <MemoryRouter initialEntries={initialEntries}>
          <Routes>
            <Route path="*" element={ui} />
            {routes.map((route) => (
              <Route key={route.path} path={route.path} element={route.element} />
            ))}
          </Routes>
        </MemoryRouter>
      </AntApp>
    </QueryClientProvider>,
  );
}
