import { ArrowRightOutlined } from '@ant-design/icons';

import { REQUEST_STATUS_META, type ExchangeRequest } from '@entities/exchangeRequest';

import { FadeInImage, StatusTag } from '@shared/ui';

import './RequestCard.scss';

interface RequestCardProps {
  request: ExchangeRequest;
  // the request doesn't return a snapshot of the given item — the page takes it from the
  // items cache by offeredItemId
  offeredItemImageUrl?: string | null;
  onClick: () => void;
}

export function RequestCard({ request, offeredItemImageUrl, onClick }: RequestCardProps) {
  const statusMeta = REQUEST_STATUS_META[request.status];

  return (
    <article
      className="request-card"
      role="button"
      tabIndex={0}
      aria-label={`Запрос: отдаю ${request.offeredItemTitle ?? 'товар'}, хочу ${request.wantedDescription}`}
      onClick={onClick}
      onKeyDown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault();
          onClick();
        }
      }}
    >
      <div className="request-card__thumb">
        {offeredItemImageUrl ? (
          <FadeInImage className="request-card__image" src={offeredItemImageUrl} alt="" />
        ) : (
          <div className="request-card__placeholder" aria-hidden />
        )}
      </div>
      <div className="request-card__col">
        {/* the given item and status are on one line; the wanted one on a second line with an
    arrow instead of captions: all together wouldn't fit one line on narrow screens */}
        <div className="request-card__head">
          <span className="request-card__offer">{request.offeredItemTitle ?? 'Товар удалён'}</span>
          <StatusTag tone={statusMeta.tone}>{statusMeta.label}</StatusTag>
        </div>
        <p className="request-card__swap">
          <ArrowRightOutlined className="request-card__arrow" aria-hidden />
          <span className="request-card__want">{request.wantedDescription}</span>
        </p>
      </div>
    </article>
  );
}
