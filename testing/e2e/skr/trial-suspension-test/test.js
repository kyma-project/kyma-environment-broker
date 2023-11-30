const {gatherOptions} = require("../skr-test/helpers");
const {getOrProvisionSKR} = require("../skr-test/provision/provision-skr");
const {KCPWrapper, KCPConfig} = require('../kcp/client');
const {deprovisionAndUnregisterSKR} = require("../skr-test/provision/deprovision-skr");

const kcp = new KCPWrapper(KCPConfig.fromEnv());

const provisioningTimeout = 1000 * 60 * 30; // 30m
const suspensionTimeout = 1000 * 60 * 60; // 60m
const deprovisioningAfterSuspensionTimeout = 1000 * 60 * 5; // 5m
const slowTestDuration = 1000 * 60 * 40; // 40m
const globalTimeout = provisioningTimeout + suspensionTimeout;

describe('SKR Trial suspension test', function () {
    this.timeout(globalTimeout);
    this.slow(slowTestDuration);

    let skr;
    let options = gatherOptions();

    before('Ensure SKR Trial is provisioned', async function() {
        this.timeout(provisioningTimeout);
        skr = await getOrProvisionSKR(options, false, provisioningTimeout);
        options = skr.options;
    });

    it('should wait until Trial Cleanup CronJob triggers suspension', async function() {

    });

    it('should wait until suspension succeeds', async function() {

    });

    after('Cleanup the resources', async function() {
        await deprovisionAndUnregisterSKR(options, deprovisioningAfterSuspensionTimeout, false, true);
    });
});