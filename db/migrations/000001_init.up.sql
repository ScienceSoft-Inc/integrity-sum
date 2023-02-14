CREATE TABLE IF NOT EXISTS hashfiles (
    id BIGSERIAL PRIMARY KEY,
    file_name VARCHAR NOT NULL,
    full_file_path TEXT NOT NULL,
    algorithm VARCHAR NOT NULL,
    hash_sum VARCHAR NOT NULL,
    name_deployment TEXT,
    name_pod TEXT,
    time_of_creation VARCHAR (50),
    image_tag TEXT
);