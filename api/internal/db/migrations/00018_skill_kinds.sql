-- +goose Up
-- Skills are hard (technical) or soft (interpersonal, learning, leadership),
-- so the dashboard's skills network can show both (EP-30).
ALTER TABLE skills ADD COLUMN kind text NOT NULL DEFAULT 'hard' CHECK (kind IN ('hard', 'soft'));
UPDATE skills s SET kind = 'soft' FROM roadmaps r JOIN entities e ON e.id = r.id
WHERE s.roadmap_id = r.id AND r.source = 'library' AND e.title = 'Soft skills for engineers';

-- +goose Down
ALTER TABLE skills DROP COLUMN kind;
