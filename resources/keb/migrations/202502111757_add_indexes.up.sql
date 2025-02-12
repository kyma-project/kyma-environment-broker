CREATE INDEX operations_by_iid_created_at ON operations USING btree (instance_id, created_at);
