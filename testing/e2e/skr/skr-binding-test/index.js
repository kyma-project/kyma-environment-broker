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
  globalTimeout += provisioningTimeout + deprovisioningTimeout;

  this.timeout(globalTimeout);
  this.slow(slowTime);

  const options = gatherOptions(); // with default values
  let kubeconfigFromBinding;
  let bindingID;

  before('Ensure SKR is provisioned', async function() {
    this.timeout(provisioningTimeout);
    await provisionSKRInstance(options, provisioningTimeout);
  });

  it('Create SKR binding using Kubernetes TokenRequest', async function() {
    bindingID = Math.random().toString(36).substring(2, 18);
    try {
      kubeconfigFromBinding = await keb.createBinding(options.instanceID, bindingID, true);
    } catch (err) {
      console.log(err);
    }
  });

  it('Initiate K8s client with kubeconfig from binding', async function() {
    await initializeK8sClient({kubeconfig: kubeconfigFromBinding.credentials.kubeconfig});
  });

  it('Fetch sap-btp-manager secret using binding from Kubernetes TokenRequest', async function() {
    await getSecret(secretName, ns);
  });

  it('Fetch SKR binding created using Kubernetes TokenRequest', async function() {
    const retrievedBinding = await keb.getBinding(options.instanceID, bindingID);
    expect(retrievedBinding.credentials.kubeconfig).to.equal(kubeconfigFromBinding.credentials.kubeconfig);
  });

  it('Delete SKR binding created using Kubernetes TokenRequest', async function() {
    await keb.deleteBinding(options.instanceID, bindingID);
  });

  it('Try to fetch SKR binding created using Kubernetes TokenRequest', async function() {
    try {
      await keb.getBinding(options.instanceID, bindingID);
      fail('KEB must return an error');
    } catch (err) {
      expect(err.response.status).equal(404);
      console.log('Got response:');
      console.log(err.response.data);
    }
  });

  it('Try to fetch sap-btp-manager secret using binding from Kubernetes TokenRequest', async function() {
    try {
      await getSecret(secretName, ns);
      fail('KCP must return an error');
    } catch (err) {
      console.log(err);
    }
  });

  it('Create SKR binding using Gardener', async function() {
    bindingID = Math.random().toString(36).substring(2, 18);
    const expirationSeconds = 900;
    try {
      kubeconfigFromBinding = await keb.createBinding(options.instanceID, bindingID, false, expirationSeconds);
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

  it('Fetch SKR binding created using Gardener', async function() {
    const retrievedBinding = await keb.getBinding(options.instanceID, bindingID);
    expect(retrievedBinding.credentials.kubeconfig).to.equal(kubeconfigFromBinding.credentials.kubeconfig);
  });

  it('Delete SKR binding created using Gardener', async function() {
    await keb.deleteBinding(options.instanceID, bindingID);
  });

  it('Try to fetch SKR binding created using Gardener', async function() {
    try {
      await keb.getBinding(options.instanceID, bindingID);
      fail('KEB must return an error');
    } catch (err) {
      expect(err.response.status).equal(404);
      console.log('Got response:');
      console.log(err.response.data);
    }
  });

  it('Try to fetch sap-btp-manager secret using binding from Gardener', async function() {
    await getSecret(secretName, ns);
  });

  after('Cleanup the resources', async function() {
    this.timeout(deprovisioningTimeout);
    if (process.env['SKIP_DEPROVISIONING'] != 'true') {
      await deprovisionAndUnregisterSKR(options, deprovisioningTimeout, true);
    }
  });
});
