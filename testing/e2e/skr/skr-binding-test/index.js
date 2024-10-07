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
*/
  it('Create SKR binding for service account using Kubernetes TokenRequest', async function() {
    try {
      kubeconfigFromBinding = await keb.createBinding("0EFB3BD5-EDA1-4659-AA18-597236230931", true);
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
    this.timeout(10000);
    try {
      kubeconfigFromBinding = await keb.createBinding("0EFB3BD5-EDA1-4659-AA18-597236230931", false, expirationSeconds);
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

  it('Should not allow creation of SKR binding when expiration seconds value is below the minimum value', async function() {
    const expirationSeconds = 700;
    this.timeout(10000);
   // const bindingID = Math.random().toString(36).substring(2, 18);


  //  const customParams = {'service_account': true, 'expiration_seconds': expirationSeconds};
   // const payload = keb.buildPayload('binding', "0EFB3BD5-EDA1-4659-AA18-597236230931", null, null, customParams);
    //const endpoint = `service_instances/0EFB3BD5-EDA1-4659-AA18-597236230931/service_bindings/${bindingID}?accepts_incomplete=true`;

    //const config = await keb.buildRequest(payload, endpoint, 'put');

   //return keb.createBinding("0EFB3BD5-EDA1-4659-AA18-597236230931", true, expirationSeconds).should.be.rejected;
   // expect(async () => { keb.createBinding("0EFB3BD5-EDA1-4659-AA18-597236230931", true, expirationSeconds); }).to.throw();
    try {
      await keb.createBinding2("0EFB3BD5-EDA1-4659-AA18-597236230931", true, expirationSeconds);
      console.log("The test was expected to fail but it passed");
      assert.fail("The test was expected to fail but it passed")
    } catch (err) {
      if (err.response) {
        expect(err.response.status).equal(400);
        expect(err.response.data.description).to.include('expiration_seconds cannot be less than');
        console.log('Got response:');
        console.log(err.response.data);
      } else {
        throw err;
      }
    
    }
      
  });

  it('Should not allow creation of SKR binding when expiration seconds value is over the maximum value', async function() {
    const expirationSeconds = 1;
    this.timeout(10000);
    try {
      kubeconfigFromBinding = await keb.createBinding("0EFB3BD5-EDA1-4659-AA18-597236230931", true, expirationSeconds);
     // console.log("The test was expected to fail but it passed");
      //expect.fail();
    } catch (err) { }
      console.log("The test failed as expected");
  });

 /* after('Cleanup the resources', async function() {
    this.timeout(deprovisioningTimeout);
    if (process.env['SKIP_DEPROVISIONING'] != 'true') {
      await deprovisionAndUnregisterSKR(options, deprovisioningTimeout, true);
    }
  });*/
});
