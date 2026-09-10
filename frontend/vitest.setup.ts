import '@testing-library/jest-dom/vitest';
import { cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';

// jsdom doesn't implement matchMedia, but antd's adaptive components rely on it;
// a real node environment has no window, so the shim is included only under jsdom
if (typeof window !== 'undefined') {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  });

  // antd (Upload, Select, etc.) relies on ResizeObserver, which jsdom lacks
  class ResizeObserverMock {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  window.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver;

  // jsdom doesn't implement the Object URL API. Without the shim, the chosen-photo preview
  // would always be null, and item-form tests would "pass" without ever checking the preview
  if (typeof URL.createObjectURL !== 'function') {
    let counter = 0;
    URL.createObjectURL = () => `blob:mock/${++counter}`;
    URL.revokeObjectURL = () => {};
  }
}

afterEach(() => {
  cleanup();
});
