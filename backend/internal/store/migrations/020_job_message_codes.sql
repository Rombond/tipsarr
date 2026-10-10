-- Job summaries are stored as a message code plus parameters, so each app can show them in its own language.
-- `message` keeps the English text (logs, old rows, clients that do not know the code).

ALTER TABLE job_runs ADD COLUMN message_code VARCHAR(80) NOT NULL DEFAULT '';
ALTER TABLE job_runs ADD COLUMN message_params TEXT NOT NULL DEFAULT '';
