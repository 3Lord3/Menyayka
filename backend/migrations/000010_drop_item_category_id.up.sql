-- The item category is now only the textual items.category field.
ALTER TABLE items DROP COLUMN IF EXISTS category_id;
