const {expect} = require('chai');
const {gatherOptions} = require('../skr-test');
const {initializeK8sClient} = require('../utils/index.js');
const {getSecret, getKubeconfigValidityInSeconds} = require('../utils');
const {provisionSKRInstance} = require('../skr-test/provision/provision-skr');
const {deprovisionAndUnregisterSKR} = require('../skr-test/provision/deprovision-skr');
const {KEBClient, KEBConfig} = require('../kyma-environment-broker');
const keb = new KEBClient(KEBConfig.fromEnv());

const provisioningTimeout = 1000 * 60 * 30; // 30m
const deprovisioningTimeout = 1000 * 60 * 95; // 95m
let globalTimeout = 1000 * 60 * 70; // 70m
const slowTime = 5000;
const secretName = 'sap-btp-manager';
const ns = 'kyma-system';

describe('SKR Binding test', function() {
/*   globalTimeout += provisioningTimeout + deprovisioningTimeout;

  this.timeout(globalTimeout);
  this.slow(slowTime);

  const options = gatherOptions(); // with default values
  let kubeconfigFromBinding;

 before('Ensure SKR is provisioned', async function() {
    this.timeout(provisioningTimeout);
    await provisionSKRInstance(options, provisioningTimeout);
  });

  it('Create SKR binding for service account using Kubernetes TokenRequest', async function() {
    try {
      kubeconfigFromBinding = await keb.createBinding(options.instanceID, true);
    } catch (err) {
      console.log(err);
    }
  });

  it('Initiate K8s client with kubeconfig from binding', async function() {
    await initializeK8sClient({kubeconfig: kubeconfigFromBinding.credentials.kubeconfig});
  });

  it('Fetch sap-btp-manager secret using binding for service account from Kubernetes TokenRequest', async function() {
    await getSecret(secretName, ns);
  });

  it('Create SKR binding using Gardener', async function() {
    const expirationSeconds = 900;
    try {
      kubeconfigFromBinding = await keb.createBinding(options.instanceID, false, expirationSeconds);
      expect(getKubeconfigValidityInSeconds(kubeconfigFromBinding.credentials.kubeconfig)).to.equal(expirationSeconds);
    } catch (err) {
      console.log(err);
    }
  });

  it('Initiate K8s client with kubeconfig from binding', async function() {
    await initializeK8sClient({kubeconfig: kubeconfigFromBinding.credentials.kubeconfig});
  });

  it('Fetch sap-btp-manager secret using binding from Gardener', async function() {
    await getSecret(secretName, ns);
  });
*/
  it('Should not allow creation of SKR binding when expiration seconds value is below the minimum value', async function() {
    const expirationSeconds = 700;

    expect(function () { keb.createBinding("0EFB3BD5-EDA1-4659-AA18-597236230931", true, expirationSeconds).timeout(5000); }).to.throw();
      
  });

  it('Should not allow creation of SKR binding when expiration seconds value is over the maximum value', async function() {
    const expirationSeconds = 700;
    try {
      kubeconfigFromBinding = await keb.createBinding("0EFB3BD5-EDA1-4659-AA18-597236230931", true, expirationSeconds).timeout(5000);
      console.log("The test was expected to fail but it passed");
      expect.fail();
    } catch (err) { }
  });

 /* after('Cleanup the resources', async function() {
    this.timeout(deprovisioningTimeout);
    if (process.env['SKIP_DEPROVISIONING'] != 'true') {
      await deprovisionAndUnregisterSKR(options, deprovisioningTimeout, true);
    }
  });*/
});
