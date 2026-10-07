-- Quality profile / root folder chosen when the request was made (0 / empty = instance default).

ALTER TABLE requests ADD COLUMN profile_id INT NOT NULL DEFAULT 0;
ALTER TABLE requests ADD COLUMN root_folder VARCHAR(512) NOT NULL DEFAULT '';
