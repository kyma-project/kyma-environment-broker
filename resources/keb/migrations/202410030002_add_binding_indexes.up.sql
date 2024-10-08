CREATE INDEX bindings_by_instance_id ON bindings USING btree (instance_id);
CREATE INDEX bindings_by_id ON bindings USING btree (id);
