-- Add agency ContentOS card fields for publish copy and revision tracking.
ALTER TABLE content_items
  ADD COLUMN IF NOT EXISTS caption TEXT DEFAULT '',
  ADD COLUMN IF NOT EXISTS revision_notes TEXT DEFAULT '';
