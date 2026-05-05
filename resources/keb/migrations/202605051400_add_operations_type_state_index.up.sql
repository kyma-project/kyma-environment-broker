CREATE INDEX operations_by_type_state_created_at ON operations USING btree (type, state, created_at);
