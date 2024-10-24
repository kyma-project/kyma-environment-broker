const {expect} = require('chai');
const {gatherOptions} = require('../skr-test');
const {initializeK8sClient} = require('../utils/index.js');
const {getSecret} = require('../utils');
const {provisionSKRInstance} = require('../skr-test/provision/provision-skr');
const {deprovisionAndUnregisterSKR} = require('../skr-test/provision/deprovision-skr');
const {KEBClient, KEBConfig} = require('../kyma-environment-broker');
const uuid = require('uuid');
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

  it('Create SKR binding', async function() {
    bindingID = uuid.v4();
    try {
      const resp = await keb.createBinding(options.instanceID, bindingID);
      kubeconfigFromBinding = resp.data.credentials.kubeconfig;
      expect(resp.status).equal(201);
    } catch (err) {
      throw err;
    }
  });

  it('Initiate K8s client with kubeconfig from binding', async function() {
    await initializeK8sClient({kubeconfig: kubeconfigFromBinding});
  });

  it('Get sap-btp-manager secret', async function() {
    await getSecret(secretName, ns);
  });

  it('Get SKR binding', async function() {
    const resp = await keb.getBinding(options.instanceID, bindingID);
    expect(resp.data.credentials.kubeconfig).to.equal(kubeconfigFromBinding);
    expect(resp.status).equal(200);
  });

  it('Delete SKR binding', async function() {
    const resp = await keb.deleteBinding(options.instanceID, bindingID);
    expect(resp.status).equal(200);

    try {
      await keb.getBinding(options.instanceID, bindingID);
      expect.fail('The call was expected to fail but it passed. Binding was retrieved after deletion');
    } catch (err) {
      if (err.response) {
        expect(err.response.status).equal(404);
        console.log('Got response:');
        console.log(err.response.data);
      } else {
        throw err;
      }
    }
  });

  it('Should not allow to get sap-btp-manager secret', async function() {
    try {
      await getSecret(secretName, ns);
      expect.fail('The call was expected to fail but it passed. Got the secret');
    } catch (err) {
      if (err.response) {
        console.log('Got response:');
        console.log(err.response.data);
      } else {
        throw err;
      }
    }
  });

  it('Should not allow creation of SKR binding when expiration seconds value is below the min value', async function() {
    bindingID = uuid.v4();
    const expirationSeconds = 1;
    try {
      await keb.createBinding(options.instanceID, bindingID, expirationSeconds);
      expect.fail('The call was expected to fail but it passed');
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

  it('Should not allow creation of SKR binding when expiration seconds value is over the max value', async function() {
    bindingID = uuid.v4();
    const expirationSeconds = 999999999;
    try {
      await keb.createBinding(options.instanceID, bindingID, expirationSeconds);
      expect.fail('The call was expected to fail but it passed');
    } catch (err) {
      if (err.response) {
        expect(err.response.status).equal(400);
        expect(err.response.data.description).to.include('expiration_seconds cannot be greater than');
        console.log('Got response:');
        console.log(err.response.data);
      } else {
        throw err;
      }
    }
  });

  it('Should return HTTP 200 when creating a binding with the same ID and params as an existing one', async function() {
    bindingID = uuid.v4();
    const expirationSeconds = 600;
    const firstResponse = await keb.createBinding(options.instanceID, bindingID, expirationSeconds);
    expect(firstResponse.status).equal(201);

    const secondResponse = await keb.createBinding(options.instanceID, bindingID, expirationSeconds);
    expect(secondResponse.status).equal(200);
  });

  it('Should return HTTP 409 for creating duplicate binding but with different params', async function() {
    const expirationSeconds = 700;
    try {
      await keb.createBinding(options.instanceID, bindingID, expirationSeconds);
      expect.fail('The call was expected to return HTTP 409');
    } catch (err) {
      if (err.response) {
        expect(err.response.status).equal(409);
        expect(err.response.data.description).to.include('binding already exists but with different parameters');
        console.log('Got response:');
        console.log(err.response.data);
      } else {
        throw err;
      }
    }
  });

  it('Should return HTTP 410 when deleting a nonexisting binding', async function() {
    try {
      const resp = await keb.deleteBinding(options.instanceID, bindingID);
      expect(resp.status).equal(200);
      await keb.deleteBinding(options.instanceID, bindingID);
      expect.fail('The call was expected to return HTTP 410');
    } catch (err) {
      if (err.response) {
        expect(err.response.status).equal(410);
        console.log('Got response:');
        console.log(err.response.data);
      } else {
        throw err;
      }
    }
  });

  it('Should return HTTP 404 when trying to get a nonexisting binding', async function() {
    try {
      await keb.getBinding(options.instanceID, bindingID);
      expect.fail('The call was expected to return HTTP 404');
    } catch (err) {
      if (err.response) {
        expect(err.response.status).equal(404);
        expect(err.response.data.description).to.include('Binding not found');
        console.log('Got response:');
        console.log(err.response.data);
      } else {
        throw err;
      }
    }
  });

  it('Should not allow creation of more than 10 SKR bindings', async function() {
    let errorOccurred = false;
    let count = 0;
    // We don't know how many bindings have been created in the previous test before we start this one.
    while (!errorOccurred && count < 13) {
      bindingID = uuid.v4();
      try {
        await keb.createBinding(options.instanceID, bindingID);
      } catch (err) {
        if (err.response) {
          errorOccurred = true;
          expect(err.response.status).equal(400);
          expect(err.response.data.description).to.include('maximum number of non expired bindings reached');
          console.log('Got response:');
          console.log(err.response.data);
        } else {
          throw err;
        }
      }
      count++;
    }

    if (count >= 13) {
      expect.fail('The call was expected to fail but it passed. Created more than 10 bindings');
    }
  });

  after('Cleanup the resources', async function() {
    this.timeout(deprovisioningTimeout);
    if (process.env['SKIP_DEPROVISIONING'] != 'true') {
      await deprovisionAndUnregisterSKR(options, deprovisioningTimeout, true);
    }
  });
});
