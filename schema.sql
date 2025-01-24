CREATE TABLE datasets (
   id INTEGER PRIMARY KEY,
   name TEXT NOT NULL UNIQUE
) STRICT;

CREATE TABLE comments (
   id INTEGER PRIMARY KEY,
   dataset_id INTEGER NOT NULL,
   comment TEXT,
   category TEXT,
   FOREIGN KEY (dataset_id) REFERENCES datasets(id)
) STRICT;

