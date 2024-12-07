CREATE TABLE photos (
    id TEXT NOT NULL PRIMARY KEY,
    file TEXT NOT NULL,
    description TEXT,
    resolution TEXT,
    taken_at TIMESTAMP WITH TIME ZONE,
    uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    modified_at TIMESTAMP WITH TIME ZONE NOT NULL,
    people_ids TEXT[],
    tags TEXT[]
);

CREATE TABLE people (
    id TEXT NOT NULL PRIMARY KEY,
    name TEXT NOT NULL,
    surname TEXT,
    notes TEXT
);
