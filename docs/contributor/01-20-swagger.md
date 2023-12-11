# Check API Using Swagger

Using the Swagger UI, you can visualize Kyma Environment Broker's (KEB's) APIs on a single page.

The Swagger UI static files are copied from the [official source](https://github.com/swagger-api/swagger-ui/tree/master/dist) and then they are injected into KEB's container which exposes them on the root endpoint.

KEB uses a [Swagger schema](https://github.com/kyma-project/kyma-environment-broker/blob/main/resources/keb/files/swagger.yaml) file that is mounted as a volume to the Pod. You can use templates in the Swagger schema file to configure it.

## Port-forward the Pod

You can port-forward the Pod to expose and use the Swagger UI. To do it, use the following command:

   ```bash
   kubectl port-forward -n kcp-system svc/kcp-kyma-environment-broker 8888:80
   ```

Open the following website:

   ```
   http://localhost:8888/
   ```
