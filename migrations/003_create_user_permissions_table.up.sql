CREATE TABLE login_attempts (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(100),
    ip_address VARCHAR(45),
    success BOOLEAN DEFAULT FALSE,
    attempt_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_login_attempts_user ON login_attempts(username, attempt_time);