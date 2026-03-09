# Check the API Using Swagger

With the Swagger UI, you can visualize Kyma Environment Broker's (KEB) APIs on a single page.

The Swagger UI static files are copied from the [official source](https://github.com/swagger-api/swagger-ui/tree/master/dist) and then injected into KEB's container which exposes them on the root endpoint.

KEB uses a [Swagger schema](https://github.com/kyma-project/kyma-environment-broker/blob/main/resources/keb/files/swagger.yaml) file mounted as a volume to the Pod. Use templates in the Swagger schema file to configure it.

## Port-Forward the Pod

To port-forward the Pod and expose and use the Swagger UI, use the following command:

   ```bash
   kubectl port-forward -n kcp-system svc/kcp-kyma-environment-broker 8888:80
   ```

<!-- markdown-link-check-disable-next-line -->
Open `http://localhost:8888/` in your browser.
