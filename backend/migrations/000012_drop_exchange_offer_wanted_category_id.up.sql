-- The wanted category is now only the textual exchange_offers.wanted_category field.
ALTER TABLE exchange_offers DROP COLUMN IF EXISTS wanted_category_id;
