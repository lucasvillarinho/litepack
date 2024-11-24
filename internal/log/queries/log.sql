-- name: CreateLogTable :exec
CREATE TABLE IF NOT EXISTS log (
    id SERIAL PRIMARY KEY,
    level TEXT NOT NULL,              
    message TEXT NOT NULL,          
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Log timestamp
);


-- name: InsertLog :exec
INSERT INTO log (level, message) VALUES (?, ?);