import { useState } from 'react';

// motion-step only on step changes: on mount the screen already plays motion-screen, the motions
// would stack
export function useStepMotionClass(step: string): string {
  const [initialStep] = useState(step);
  return step === initialStep ? '' : ' motion-step';
}
