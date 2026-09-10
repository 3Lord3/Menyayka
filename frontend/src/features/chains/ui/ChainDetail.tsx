import { theme, Button } from 'antd';

import {
  approvedVotes,
  chainLinks,
  CONFIRM_VOTE_META,
  confirmVoteAt,
  hasDeal,
  isAssembled,
  myConfirmVote,
  myParticipant,
  needsMyAction,
  participantAlias,
  receivesItem,
  showsScore,
  VACANCY_META,
  VOTE_META,
  type Chain,
  type ChainLink,
  type ChainParticipant,
  type VoteValue,
} from '@entities/chain';

import { plural } from '@shared/lib/plural';
import { Avatar, FadeInImage, ProbabilityBadge } from '@shared/ui';

import { ConsentBadge } from './ConsentBadge';

import './ChainDetail.scss';

interface ChainDetailProps {
  chain: Chain;
  // the request of the chosen receive option: the source link is shown only by it
  receiveRequestId?: number;
  onConfirm: () => void;
  onProceed: () => void;
}

// can't respond from here: only response statuses and the second-round action live here
export function ChainDetail({ chain, receiveRequestId, onConfirm, onProceed }: ChainDetailProps) {
  const { token } = theme.useToken();
  // the source link narrows to the chosen variant: its candidates are requests of different
  // users, while a link row draws a single participant
  const links = chainLinks(chain).map((link) =>
    link.position === chain.receivesFromPosition
      ? { ...link, candidates: receivesItem(chain, receiveRequestId) }
      : link,
  );
  const me = myParticipant(chain);
  const canVote = chain.status === 'CANDIDATE';
  const assembled = isAssembled(chain.status);
  const approved = approvedVotes(chain);
  // a vote is tied to the voting target, not the voter: position p's decision lives in the
  // vote of position (p + 1) % length (SOFT-LOCK §3.3); on CANDIDATE there's no shift — these
  // are responses
  const showConfirmPills = chain.status !== 'CANDIDATE';

  return (
    <div className="chain-detail">
      {assembled ? (
        <div className="chain-detail__head">
          {showsScore(chain.status) ? <ProbabilityBadge score={chain.score} /> : null}
          <ConsentBadge key={approved} count={approved} total={chain.length} />
        </div>
      ) : null}
      {chain.status === 'PROPOSED' && myConfirmVote(chain) === 'approved' ? (
        <p className="chain-detail__confirmed" role="status">
          Вы подтвердили · ждём остальных
        </p>
      ) : null}
      <ul className="chain-detail__participants motion-cascade">
        {links.map((link) => (
          <ChainLinkRow
            key={link.position}
            link={link}
            isMine={me?.position === link.position}
            isReceiveLink={link.position === chain.receivesFromPosition}
            canVote={canVote}
            confirmVote={showConfirmPills ? confirmVoteAt(chain, link.position) : undefined}
          />
        ))}
      </ul>
      {needsMyAction(chain) ? (
        <Button
          className="chain-detail__action"
          type="primary"
          size="large"
          block
          onClick={onConfirm}
        >
          Требуются действия
        </Button>
      ) : hasDeal(chain.status) ? (
        <Button
          className="chain-detail__action"
          size="large"
          block
          style={{
            backgroundColor: token.colorSuccess,
            borderColor: token.colorSuccess,
            color: '#FFFFFF',
          }}
          onClick={onProceed}
        >
          Перейти к сделке
        </Button>
      ) : null}
    </div>
  );
}

interface ChainLinkRowProps {
  link: ChainLink;
  isMine: boolean;
  isReceiveLink: boolean;
  canVote: boolean;
  confirmVote?: VoteValue | null;
}

function ChainLinkRow({ link, isMine, isReceiveLink, canVote, confirmVote }: ChainLinkRowProps) {
  const alias = participantAlias(link.position);
  const label = isMine ? 'Вы' : alias.name;
  const { candidates } = link;

  return (
    <li className={`chain-detail__participant${isMine ? ' chain-detail__participant--me' : ''}`}>
      <div className="chain-detail__participant-head">
        <Avatar
          name={label}
          size="md"
          label={isMine ? 'Я' : undefined}
          emoji={isMine ? undefined : alias.emoji}
        />
        <span className="chain-detail__participant-name">{label}</span>
        {confirmVote !== undefined ? <ConfirmVotePill vote={confirmVote} /> : null}
      </div>

      {candidates.length === 1 ? (
        <ChainLinkItem participant={candidates[0]} canVote={canVote} />
      ) : isReceiveLink ? (
        <ul className="chain-detail__candidates">
          {candidates.map((candidate) => (
            <li key={candidate.requestId} className="chain-detail__candidate">
              <ChainLinkItem participant={candidate} canVote={canVote} />
            </li>
          ))}
        </ul>
      ) : (
        <span className="chain-detail__collapsed">
          {candidates.length} {plural(candidates.length, ['вариант', 'варианта', 'вариантов'])}
        </span>
      )}
    </li>
  );
}

// null — a vacancy after a decline
function ConfirmVotePill({ vote }: { vote: VoteValue | null }) {
  const meta = vote === null ? VACANCY_META : CONFIRM_VOTE_META[vote];
  return (
    <span className={`chain-detail__confirm chain-detail__confirm--${meta.tone}`}>
      {meta.label}
    </span>
  );
}

interface ChainLinkItemProps {
  participant: ChainParticipant;
  canVote: boolean;
}

// on an assembled chain first-round pills aren't shown — there vote has a different meaning
function ChainLinkItem({ participant, canVote }: ChainLinkItemProps) {
  const voteMeta = canVote && participant.vote ? VOTE_META[participant.vote] : null;

  return (
    <div className="chain-detail__item">
      <span
        className={`chain-detail__thumb${participant.imageUrl ? '' : ' chain-detail__thumb--empty'}`}
        aria-hidden
      >
        {participant.imageUrl ? (
          <FadeInImage className="chain-detail__thumb-img" src={participant.imageUrl} alt="" />
        ) : null}
      </span>
      <div className="chain-detail__item-info">
        <span className="chain-detail__item-title">{participant.offeredItemTitle}</span>
        <span className="chain-detail__item-wanted">Хочет: {participant.wantedDescription}</span>
      </div>
      {voteMeta ? (
        <span className={`chain-detail__response chain-detail__response--${voteMeta.tone}`}>
          {voteMeta.label}
        </span>
      ) : null}
    </div>
  );
}
