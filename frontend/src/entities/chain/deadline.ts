// null — no deadline, it's invalid, or it already passed: show nothing, the backend does the
// rollback lazily itself
export function formatRemaining(deadlineAt: string | null | undefined, now: number): string | null {
  const deadline = Date.parse(deadlineAt ?? '');
  if (Number.isNaN(deadline) || deadline <= now) return null;
  const totalMinutes = Math.floor((deadline - now) / 60_000);
  const hours = Math.floor(totalMinutes / 60);
  const minutes = totalMinutes % 60;
  // exactly-with-hours we write without "0 min" — "1 h 0 min" reads as a layout bug
  if (hours > 0) return minutes > 0 ? `${hours} ч ${minutes} мин` : `${hours} ч`;
  if (minutes > 0) return `${minutes} мин`;
  return 'меньше минуты';
}
