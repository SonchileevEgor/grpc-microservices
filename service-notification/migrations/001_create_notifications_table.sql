CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    task_id INT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
