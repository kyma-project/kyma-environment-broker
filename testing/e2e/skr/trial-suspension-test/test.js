const {KCPWrapper, KCPConfig} = require('../kcp/client');
const {KEBClient, KEBConfig} = require('../kyma-environment-broker');
const {gatherOptions} = require('../skr-test/helpers');
const {getOrProvisionSKR} = require('../skr-test/provision/provision-skr');
const {ensureOperationSucceeded} = require('../kyma-environment-broker/helpers');
const {deprovisionAndUnregisterSKR} = require('../skr-test/provision/deprovision-skr');
const {debug} = require('../utils');

const kcp = new KCPWrapper(KCPConfig.fromEnv());
const keb = new KEBClient(KEBConfig.fromEnv());

const provisioningTimeout = 1000 * 60 * 30; // 30m
const suspensionTimeout = 1000 * 60 * 60; // 60m
const deprovisioningAfterSuspensionTimeout = 1000 * 60 * 5; // 5m
const trialCleanupTriggerTimeout = 1000 * 60 * 11; // 11m
const slowTestDuration = 1000 * 60 * 40; // 40m
const globalTimeout = provisioningTimeout + suspensionTimeout;
const suspensionOperationType = 'suspension';
const inProgressOperationState = 'in progress';

describe('SKR Trial suspension test', function() {
  this.timeout(globalTimeout);
  this.slow(slowTestDuration);

  let suspensionOpID;
  const options = gatherOptions();

  before('Ensure SKR Trial is provisioned', async function() {
    await getOrProvisionSKR(options, false, provisioningTimeout);
  });

  it('should wait until Trial Cleanup CronJob triggers suspension', async function() {
    const rs = await kcp.ensureLatestGivenOperationTypeIsInGivenState(options.instanceID,
        suspensionOperationType, inProgressOperationState, trialCleanupTriggerTimeout);
    suspensionOpID = rs.data[0].status[suspensionOperationType].data[0].operationID;
    assert.isDefined(suspensionOpID, `suspension operation ID: ${suspensionOpID}`);
  });

  it('should wait until suspension succeeds', async function() {
    debug(`Waiting until suspension operation succeeds...`);
    await ensureOperationSucceeded(keb, kcp, options.instanceID, suspensionOpID, suspensionTimeout);
  });

  after('Cleanup the resources', async function() {
    await deprovisionAndUnregisterSKR(options, deprovisioningAfterSuspensionTimeout, false, true);
  });
});
