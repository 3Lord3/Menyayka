import { useEffect } from 'react';

import { useLocation, useNavigationType, type NavigationType } from 'react-router';

// the session key stack is module-level: screens animating their entry have their own mount,
// but the history is shared
let historyStack: string[] = [];

function resolveDirection(
  navigationType: NavigationType,
  locationKey: string,
): 'none' | 'forward' | 'back' {
  const stack = historyStack;
  if (stack.length === 0) return 'none';
  if (navigationType === 'POP' && stack.length > 1 && stack[stack.length - 2] === locationKey) {
    return 'back';
  }
  return 'forward';
}

// we update the stack in an effect: the navigation render computes direction from the state
// before it, and the StrictMode render runs twice
function commitHistory(navigationType: NavigationType, locationKey: string): void {
  const stack = historyStack;
  if (stack[stack.length - 1] === locationKey) return;
  if (stack.length === 0) {
    historyStack = [locationKey];
    return;
  }
  if (navigationType === 'POP' && stack.length > 1 && stack[stack.length - 2] === locationKey) {
    historyStack = stack.slice(0, -1);
  } else if (navigationType === 'REPLACE') {
    historyStack = [...stack.slice(0, -1), locationKey];
  } else {
    historyStack = [...stack, locationKey];
  }
}

// POP is both a cold start and "Back"/"Forward"; direction is derived from the key stack:
// empty stack — first mount (no animation), POP to the previous key — "Back", otherwise — "Forward"
export function useScreenMotionClass(): string {
  const navigationType = useNavigationType();
  const location = useLocation();

  const direction = resolveDirection(navigationType, location.key);

  useEffect(() => {
    commitHistory(navigationType, location.key);
  }, [navigationType, location.key]);

  if (direction === 'none') return '';
  return direction === 'back' ? 'motion-screen motion-screen--back' : 'motion-screen';
}

// module-level singleton — tests need a reset between cases
export function resetScreenMotionHistory(): void {
  historyStack = [];
}
