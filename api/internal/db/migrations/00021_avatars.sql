-- +goose Up
-- Profile pictures (EP-33): a 256×256 PNG re-encoded on the server, so no
-- metadata (EXIF location, camera) survives. NULL means the generated default.
ALTER TABLE users ADD COLUMN avatar bytea CHECK (length(avatar) <= 1048576);
ALTER TABLE users ADD COLUMN avatar_updated_at timestamptz;

-- +goose Down
ALTER TABLE users DROP COLUMN avatar_updated_at;
ALTER TABLE users DROP COLUMN avatar;
