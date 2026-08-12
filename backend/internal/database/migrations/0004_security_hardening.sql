ALTER TABLE url_info DROP CONSTRAINT IF EXISTS url_info_user_id_fkey;
ALTER TABLE url_info
    ADD CONSTRAINT url_info_user_id_fkey
    FOREIGN KEY (user_id) REFERENCES user_info(user_id) ON DELETE CASCADE;

DELETE FROM session_info;
