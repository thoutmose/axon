-- +goose Up
-- What the scheduler predicted at each review of a card in long-term memory
-- (EP-77), so its calibration can be shown: predicted against remembered.
ALTER TABLE card_reviews ADD COLUMN predicted real CHECK (predicted BETWEEN 0 AND 1);

-- +goose Down
ALTER TABLE card_reviews DROP COLUMN predicted;
