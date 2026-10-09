import { Skeleton } from 'antd';
import { Navigate, useNavigate, useParams } from 'react-router';

import { DealDoneView, DealShipView, DealTransitView } from '@features/deal';

import { dealState, useChain } from '@entities/chain';

import { useRealtimeChain } from '@shared/realtime';
import { ErrorState } from '@shared/ui';

import { ChainPageHeader } from './ChainPageHeader';

import './ChainDetailPage.scss';

// before freezing there's no deal yet — a screen opened via a direct link redirects to chain details
export function DealPage() {
  const { chainId: chainIdParam } = useParams<{ chainId: string }>();
  const navigate = useNavigate();
  const chainId = chainIdParam ? Number(chainIdParam) : undefined;
  useRealtimeChain(chainId);
  const { data: chain, isLoading, isError, refetch } = useChain(chainId);
  const state = chain ? dealState(chain) : null;

  const goBack = () => navigate(`/chains/${chainId}`);
  // there's no separate link to the statuses anymore — "View chain details" leads straight to them
  const goToShipments = () => navigate(`/chains/${chainId}/deal/shipments`);
  const goToReceipts = () => navigate(`/chains/${chainId}/deal/receipts`);

  if (isLoading) {
    return (
      <div className="chain-detail-page">
        <ChainPageHeader title="Сделка" onBack={goBack} />
        <div className="chain-detail-page__body">
          <Skeleton active paragraph={{ rows: 8 }} />
        </div>
      </div>
    );
  }

  if (isError || !chain || !state) {
    return (
      <div className="chain-detail-page">
        <ChainPageHeader title="Сделка" onBack={goBack} />
        <div className="chain-detail-page__body">
          <ErrorState onRetry={refetch} />
        </div>
      </div>
    );
  }

  if (state.status === 'unavailable') {
    return <Navigate to={`/chains/${chainId}`} replace />;
  }

  return (
    <div className="chain-detail-page">
      <ChainPageHeader title="Сделка" onBack={goBack} />
      <div className="chain-detail-page__body">
        {state.status === 'ship' ? (
          <DealShipView chain={chain} deadlineAt={state.deadlineAt} onOpenDetails={goToShipments} />
        ) : state.status === 'shipped-waiting' || state.status === 'in-transit' ? (
          <DealTransitView chain={chain} state={state} onOpenDetails={goToShipments} />
        ) : (
          <DealDoneView
            state={state}
            chainId={chain.id}
            onOpenDetails={goToReceipts}
            onGoToRequests={() => navigate('/exchange-requests')}
          />
        )}
      </div>
    </div>
  );
}
