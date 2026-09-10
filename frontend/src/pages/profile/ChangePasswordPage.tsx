import { ChangePasswordFlow } from '@features/passwordChange';

import { useScreenMotionClass } from '@shared/lib/useScreenMotion';

// the screen renders outside AppLayout, so the enter transition animation is attached here
export function ChangePasswordPage() {
  const screenMotionClass = useScreenMotionClass();

  return (
    <div className={screenMotionClass}>
      <ChangePasswordFlow />
    </div>
  );
}
