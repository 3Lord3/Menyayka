-- Guarantee: an item can be an offer in at most one
-- "locked" request (IN_PROPOSAL / LOCKED). In ACTIVE status the same item may
-- appear in several requests — availability is decided at soft lock time, so
-- ACTIVE is not part of the index.
CREATE UNIQUE INDEX ux_exchange_offers_locked_in_proposal_item
    ON exchange_offers (offered_item_id)
    WHERE status IN ('LOCKED', 'IN_PROPOSAL');

-- Persist raw per-participant score features: EdgeCosines, Reliability,
-- and ClusterSizes no longer live only in ChainDraft, so the Ranker can
-- recompute chains.score after a response/withdrawal without scanning drafts.
ALTER TABLE chain_participants
    ADD COLUMN edge_cosine  DOUBLE PRECISION,
    ADD COLUMN reliability  DOUBLE PRECISION,
    ADD COLUMN cluster_size INT;