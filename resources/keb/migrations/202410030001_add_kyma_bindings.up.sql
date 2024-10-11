CREATE TABLE IF NOT EXISTS bindings (
    id VARCHAR(255) NOT NULL,
    instance_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    -- represents algorithm used to generate a kubeconfig
    binding_type VARCHAR(64) NOT NULL, 
    -- content of the kubeconfig
	kubeconfig TEXT, 
    -- expiration seconds
    expiration_seconds INTEGER,
    -- allow for the same binding id to be used for different runtimes
    PRIMARY KEY(id, instance_id)
);
