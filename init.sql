
CREATE TABLE IF NOT EXISTS files  (
    file_id SERIAL PRIMARY KEY,  -- Auto-generated ID for each file
    file_name TEXT
);

CREATE TABLE file_segments (
    segment_id SERIAL PRIMARY KEY,  -- Auto-generated ID for each segment
    file_id INT REFERENCES files(file_id),
    file_name TEXT,
    file_data BYTEA
);