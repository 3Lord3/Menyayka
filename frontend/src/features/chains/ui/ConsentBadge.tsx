import './ConsentBadge.scss';

export interface ConsentBadgeProps {
  count: number;
  total: number;
  className?: string;
}

// on FROZEN the counter is always M/M; the value comes as text, not via a single color
export function ConsentBadge({ count, total, className }: ConsentBadgeProps) {
  return (
    <span
      className={`consent-badge${className ? ` ${className}` : ''}`}
      aria-label={`${count} из ${total} согласий`}
    >
      {count}/{total} согласий
    </span>
  );
}
