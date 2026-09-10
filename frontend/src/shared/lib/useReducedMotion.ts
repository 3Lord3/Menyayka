import { useSyncExternalStore } from 'react';

const QUERY = '(prefers-reduced-motion: reduce)';

function subscribe(onChange: () => void): () => void {
  const media = window.matchMedia(QUERY);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}

function getSnapshot(): boolean {
  return window.matchMedia(QUERY).matches;
}

// An external store, not useState with an effect: the value is read synchronously in the very
// first render, so the animation can't play before the setting is accounted for
export function useReducedMotion(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot);
}
