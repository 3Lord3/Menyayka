-- Textual category of the wanted item in a request (e.g. "Phones").
ALTER TABLE exchange_offers ADD COLUMN wanted_category VARCHAR(100);
