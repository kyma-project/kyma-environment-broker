
ALTER TABLE api_definitions
DROP CONSTRAINT api_definitions_tenant_id_fkey1;
ALTER TABLE api_runtime_auths
DROP CONSTRAINT api_runtime_auths_tenant_id_fkey;
ALTER TABLE applications
DROP CONSTRAINT applications_tenant_id_fkey;
ALTER TABLE documents
DROP CONSTRAINT documents_tenant_id_fkey1;
ALTER TABLE event_api_definitions
DROP CONSTRAINT event_api_definitions_tenant_id_fkey1;
ALTER TABLE fetch_requests
DROP CONSTRAINT fetch_requests_tenant_id_fkey3;
ALTER TABLE label_definitions
DROP CONSTRAINT label_definitions_tenant_id_fkey;
ALTER TABLE labels
DROP CONSTRAINT labels_tenant_id_fkey2;
ALTER TABLE runtimes
DROP CONSTRAINT runtimes_tenant_id_fkey;
ALTER TABLE system_auths
DROP CONSTRAINT system_auths_tenant_id_fkey2;
ALTER TABLE webhooks
DROP CONSTRAINT webhooks_tenant_id_fkey1;