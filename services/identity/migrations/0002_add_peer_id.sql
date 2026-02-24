-- Add peer_id to nodes table
ALTER TABLE nodes ADD COLUMN IF NOT EXISTS peer_id TEXT;
CREATE INDEX IF NOT EXISTS idx_nodes_peer_id ON nodes(peer_id);
