import type { ChainStatus } from '@entities/chain';

import { useDeadlineLabel, type DeadlinePurpose } from '../model/useDeadlineLabel';

import './DeadlineRow.scss';

interface DeadlineRowProps {
  status: ChainStatus;
  deadlineAt?: string | null;
  // the shipping deadline shows only the options-list card
  showShipDeadline?: boolean;
  // takes precedence over status-based auto-detection (see chainDeadlinePurpose)
  purpose?: DeadlinePurpose;
}

// renders null without a label so calling screens don't duplicate the condition; no role="status"
// — a live region updated every 30 s would keep interrupting the screen reader
export function DeadlineRow({ status, deadlineAt, showShipDeadline, purpose }: DeadlineRowProps) {
  const auto: DeadlinePurpose | null =
    status === 'PROPOSED' ? 'response' : showShipDeadline && status === 'FROZEN' ? 'ship' : null;
  const resolved = purpose ?? auto;
  const label = useDeadlineLabel(resolved, deadlineAt);
  if (!label) return null;
  const suffix =
    resolved === 'ship' ? 'на отправку' : resolved === 'replacement' ? 'на замену' : 'на ответ';
  return (
    <p className="deadline-row">
      <span className="deadline-row__icon" aria-hidden>
        ⏱
      </span>
      Осталось {label} {suffix}
    </p>
  );
}
