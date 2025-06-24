CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$
    DECLARE
        i INT := 0;
        v_instance_id UUID;
        v_runtime_id UUID;
        v_global_account_id UUID;
        v_sub_account_id UUID;
        v_operation_id UUID;
        v_operation_update_id UUID;
        v_data JSONB := '{"plan_id":"361c511f-f939-4621-b228-d0fb79a1fe15","service_id":"47c9dcbf-ff30-448e-ab36-d3bad66ba281","parameters":{"autoScalerMin":3,"autoScalerMax":20,"region":"eu-central-1"},"platform_provider":"AWS"}'::jsonb;
    BEGIN
        WHILE i < 1000 LOOP
            v_instance_id := gen_random_uuid();
            v_runtime_id := gen_random_uuid();
            v_global_account_id := gen_random_uuid();
            v_sub_account_id := gen_random_uuid();
            v_operation_id := gen_random_uuid();
            v_operation_update_id := gen_random_uuid();

            INSERT INTO instances (
                instance_id,
                runtime_id,
                global_account_id,
                service_id,
                service_plan_id,
                dashboard_url,
                provisioning_parameters,
                sub_account_id,
                service_name,
                service_plan_name,
                provider_region,
                version,
                provider,
                subscription_global_account_id
            ) VALUES (
                v_instance_id,
                v_runtime_id,
                v_global_account_id,
                '47c9dcbf-ff30-448e-ab36-d3bad66ba281',
                '361c511f-f939-4621-b228-d0fb79a1fe15',
                format('https://dashboard.dev.kyma.cloud.sap/?kubeconfigID=%s', v_instance_id),
                v_data::text,
                v_sub_account_id,
                'kymaruntime',
                'aws',
                'eu-central-1',
                2,
                'AWS',
                v_global_account_id
            );

            INSERT INTO operations (
                id,
                instance_id,
                target_operation_id,
                version,
                state,
                description,
                type,
                data,
                created_at,
                updated_at,
                provisioning_parameters,
                finished_stages
            ) VALUES (
                v_operation_id,
                v_instance_id,
                '',
                1,
                'succeeded',
                'Processing finished',
                'provision',
                v_data,
                now(),
                now(),
                v_data,
                'start,create_runtime,check_kyma,create_kyma_resource'
            );

            INSERT INTO operations (
                id,
                instance_id,
                target_operation_id,
                version,
                state,
                description,
                type,
                data,
                created_at,
                updated_at,
                provisioning_parameters,
                finished_stages
            ) VALUES (
                v_operation_update_id,
                v_instance_id,
                '',
                1,
                'succeeded',
                'Processing finished',
                'update',
                v_data,
                now(),
                now(),
                v_data,
                'cluster,btp-operator,btp-operator-check,check,runtime_resource,check_runtime_resource'
            );

            i := i + 1;
        END LOOP;
    END
$$;

UPDATE instances i
SET last_operation_id = o.id
FROM (
         SELECT DISTINCT ON (instance_id)
             id,
             instance_id,
             created_at
         FROM operations
         WHERE type = 'update'
         ORDER BY instance_id, created_at DESC
     ) o
WHERE i.instance_id = o.instance_id;
