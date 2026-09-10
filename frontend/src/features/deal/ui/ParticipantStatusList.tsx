import {
  myParticipant,
  nextInRing,
  participantAlias,
  type Chain,
  type ChainParticipant,
} from '@entities/chain';

import { Avatar } from '@shared/ui';

import './ParticipantStatusList.scss';

export type DealStatusMode = 'shipments' | 'receipts';

interface ParticipantStatusListProps {
  chain: Chain;
  mode: DealStatusMode;
}

// position p receives the item of the next ring position — this is what the receipt status
// is built on
export function ParticipantStatusList({ chain, mode }: ParticipantStatusListProps) {
  const me = myParticipant(chain);
  const participants = [...chain.participants].sort((a, b) => a.position - b.position);

  return (
    <ul className="participant-status motion-cascade">
      {participants.map((participant) => (
        <ParticipantStatusRow
          key={participant.requestId}
          chain={chain}
          participant={participant}
          mode={mode}
          isMe={me?.position === participant.position}
        />
      ))}
    </ul>
  );
}

function sourceFor(chain: Chain, position: number): ChainParticipant | null {
  const next = nextInRing(chain, position);
  if (next === null) return null;
  return chain.participants.find((p) => p.position === next) ?? null;
}

function ParticipantStatusRow({
  chain,
  participant,
  mode,
  isMe,
}: {
  chain: Chain;
  participant: ChainParticipant;
  mode: DealStatusMode;
  isMe: boolean;
}) {
  const alias = participantAlias(participant.position);
  const source = sourceFor(chain, participant.position);
  // a text caption is mandatory — the status isn't conveyed by color alone
  const pill =
    mode === 'shipments'
      ? participant.requestStatus === 'LOCKED'
        ? { label: 'Ожидает отправки', tone: 'warning' as const }
        : { label: 'Отправлено', tone: 'success' as const }
      : source?.requestStatus === 'DONE'
        ? { label: 'Получил', tone: 'success' as const }
        : { label: 'Ожидаем', tone: 'warning' as const };

  return (
    <li
      className={`participant-status__row${isMe ? ' participant-status__row--me' : ''}`}
      aria-label={isMe ? 'Вы' : alias.name}
    >
      <div className="participant-status__head">
        <Avatar
          name={isMe ? 'Вы' : alias.name}
          size="md"
          label={isMe ? 'Я' : undefined}
          emoji={isMe ? undefined : alias.emoji}
        />
        <span className="participant-status__name">{isMe ? 'Вы' : alias.name}</span>
        <span
          key={pill.label}
          className={`participant-status__pill participant-status__pill--${pill.tone}`}
        >
          {pill.label}
        </span>
      </div>
      <p className="participant-status__swap">
        {participant.offeredItemTitle} → {source?.offeredItemTitle ?? 'Товар'}
      </p>
    </li>
  );
}
