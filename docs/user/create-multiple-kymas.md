# Create Kyma Instances

Set up a Kubernetes cluster with SAP BTP, Kyma runtime and use it to build applications and extensions to your SAP and third-party solutions.
You can create one or multiple Kyma clusters in a single SAP BTP subaccount.

## Prerequisites

Your subaccount has entitlements for Kyma runtime configured. See [Configure Entitlements and Quotas for Subaccounts](https://help.sap.com/docs/btp/sap-business-technology-platform/configure-entitlements-and-quotas-for-subaccounts?version=Cloud).

## Context

To set up Kyma environment in your subaccount, you must create an instance of it. You can create it from your subaccount **Overview** section by choosing **Enable Kyma** and following the wizard steps to configure the provisioning parameters.
You can also create a Kyma environment instance from Service Marketplace in the same way as any other SAP BTP service or application.

If you prefer to work in a terminal or want to automate operations using scripts, you can create the Kyma environment with the SAP BTP command line interface (btp CLI). See [Enable SAP BTP, Kyma Runtime Using the Command Line](https://developers.sap.com/tutorials/btp-cli-setup-kyma-cluster.html?locale=en-US).

> [!NOTE]
> To indicate that your SAP BTP, Kyma runtime is used for production, select **Used for production** in your subaccount details. With this setting, Kyma runtime operators prioritize incidents and support cases affecting production subaccounts over subaccounts used for non-production purposes. See [Change Subaccount Details](https://help.sap.com/docs/btp/sap-business-technology-platform/change-subaccount-details?locale=en-US&version=Cloud).

## Procedure

1. In the SAP BTP cockpit, navigate to your subaccount **Overview**.
2. In the Kyma Environment section, choose **Enable Kyma** when creating your first Kyma instance. When provisioning the second and subsequent Kyma clusters, choose **Create**.
3. In the **Basic Info** view of the wizard window, perform the following actions:
   
    - Choose one of the plans assigned to your account.
    - Change the instance and cluster names or keep the default ones.
    - Choose a region from the list.

4. To continue the configuration, choose **Next**.
5. In the **Additional Parameters** view, provide the required details in the **Form**. Alternatively, switch to the **JSON** tab and upload your configuration file or specify the parameters in JSON format.
   For more information on the configurable parameters, see [Provisioning and Updating Parameters in the Kyma Environment](https://help.sap.com/docs/btp/sap-business-technology-platform/provisioning-and-update-parameters-in-kyma-environment?locale=en-US&version=Cloud).
6. To review your configuration, choose **Next**.
7. To confirm changes, choose **Create**.
8. Wait until the instance is created. The instance creation may take several minutes. In the Kyma Environment section of your subaccount **Overview**, you can monitor the instance creation status and view the operation details.
   > [!NOTE]
   > If the Kyma instances quota assigned to your subaccount has been used up, you get a message informing you about it. To increase the quota, go to **Entitlements** and make the necessary changes or contact your administrator.

## Results

You have created a Kyma environment instance. If needed, you can repeat the procedure to create another one.

In the process of provisioning a Kyma cluster, an instance of SAP Service Manager is created (see [Preconfigured Credentials and Access](https://help.sap.com/docs/btp/sap-business-technology-platform/preconfigured-credentials-and-access?locale=en-US&version=Cloud)). Therefore, each Kyma instance has an instance of Service Manager assigned to it. If you have multiple Kyma clusters in your subaccount, you can easily trace which Service Manager instance is assigned to which Kyma instance by the Service Manager instance's display name, which is the same as its matching Kyma instance's ID.

> [!NOTE]
> If the creation of an SAP Service Manager instance doesn't succeed, Kyma provisioning cannot begin, and you get an error message.

In the Kyma Environment section of your subaccount Overview, you can see all the Kyma instances created in your subaccount. If there is more than one, choose an instance from the dropdown list to view the following details:

- Kyma instance ID
- Kyma instance name
- Plan
- Kyma dashboard link
- Kubeconfig link
- Cluster name 

##  Next Steps

To view the details of your Kyma instance, choose it from the list under **Instances and Subscriptions**.

To manage your Kyma instance, go to **Instances and Subscriptions**. Choose the action you want to perform from the instance's actions menu.

To manage access to the Kyma environment and Kyma dashboard, assign roles as needed. See [Assign Roles in the Kyma Environment](https://help.sap.com/docs/btp/sap-business-technology-platform/preconfigured-credentials-and-access?locale=en-US&version=Cloud).

To use a variety of functionalities, such as telemetry and eventing, or to use SAP BTP services, add the respective Kyma modules. See [Kyma Modules](https://help.sap.com/docs/btp/sap-business-technology-platform/kyma-modules?locale=en-US&version=Cloud).

> [!TIP ]
> To manage a Kyma instance automatically, you can create a Kyma service binding. The binding enables getting a Kyma kubeconfig, which in turn allows for accessing a Kyma cluster, deploying applications, running tests, and deleting the resources in a fully automated way. See [Managing Kyma Runtime Using the Provisioning Service API](https://help.sap.com/docs/btp/sap-business-technology-platform/managing-kyma-runtime-using-provisioning-service-api?locale=en-US&version=Cloud).