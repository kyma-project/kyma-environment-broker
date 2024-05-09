package broker

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
)

const (
	kubeconfigURLKey       = "KubeconfigURL"
	notExpiredInfoFormat   = "Your cluster expires %s."
	trialExpiryDetailsKey  = "Trial account expiration details"
	trialDocsKey           = "Trial account documentation"
	trialExpireDuration    = time.Hour * 24 * 14
	trialExpiredInfoFormat = "Your cluster has expired. It is not operational and the link to Kyma dashboard is no longer valid." +
		" To continue using Kyma, you must create a new cluster. To learn how, follow the link to the trial account documentation."
	freeExpiryDetailsKey  = "Free plan expiration details"
	freeDocsKey           = "Available plans documentation"
	freeExpiredInfoFormat = "Your cluster has expired. It is not operational, and the link to Kyma dashboard is no longer valid." +
		"  To continue using Kyma, you must use a paid service plan. To learn more about the available plans, follow the link to the documentation."
)

func ResponseLabels(op internal.ProvisioningOperation, instance internal.Instance, brokerURL string, enableKubeconfigLabel bool) map[string]string {
	brokerURL = strings.TrimLeft(brokerURL, "https://")
	brokerURL = strings.TrimLeft(brokerURL, "http://")

	responseLabels := make(map[string]string, 0)
	responseLabels["Name"] = op.ProvisioningParameters.Parameters.Name
	if enableKubeconfigLabel && !IsOwnClusterPlan(instance.ServicePlanID) {
		responseLabels[kubeconfigURLKey] = fmt.Sprintf("https://%s/kubeconfig/%s", brokerURL, instance.InstanceID)
	}

	return responseLabels
}

func ResponseLabelsWithExpirationInfo(
	op internal.ProvisioningOperation,
	instance internal.Instance,
	brokerURL string,
	docsURL string,
	enableKubeconfigLabel bool,
	docsKey string,
	expireDuration time.Duration,
	expiryDetailsKey string,
	expiredInfoFormat string,
) map[string]string {
	labels := ResponseLabels(op, instance, brokerURL, enableKubeconfigLabel)

	expireTime := instance.CreatedAt.Add(expireDuration)
	hoursLeft := calculateHoursLeft(expireTime)
	if instance.IsExpired() {
		delete(labels, kubeconfigURLKey)
		labels[expiryDetailsKey] = expiredInfoFormat
		labels[docsKey] = docsURL
	} else {
		if hoursLeft < 0 {
			hoursLeft = 0
		}
		daysLeft := math.Round(hoursLeft / 24)
		switch {
		case daysLeft == 0:
			labels[expiryDetailsKey] = fmt.Sprintf(notExpiredInfoFormat, "today")
		case daysLeft == 1:
			labels[expiryDetailsKey] = fmt.Sprintf(notExpiredInfoFormat, "tomorrow")
		default:
			daysLeftNotice := fmt.Sprintf("in %2.f days", daysLeft)
			labels[expiryDetailsKey] = fmt.Sprintf(notExpiredInfoFormat, daysLeftNotice)
		}
	}

	return labels
}

func calculateHoursLeft(expireTime time.Time) float64 {
	timeLeftUntilExpire := time.Until(expireTime)
	timeLeftUntilExpireRoundedToHours := timeLeftUntilExpire.Round(time.Hour)
	return timeLeftUntilExpireRoundedToHours.Hours()
}
