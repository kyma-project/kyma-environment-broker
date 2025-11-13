package broker

import (
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/kyma-project/kyma-environment-broker/internal/kubeconfig"
)

const (
	kubeconfigURLKey       = "KubeconfigURL"
	apiServerURLKey        = "APIServerURL"
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
	apiServerURLErrorFormat = "while getting APIServerURL: %s"
)

func ResponseLabels(instance internal.Instance, brokerURL string, kubeconfigBuilder kubeconfig.KcBuilder) map[string]any {
	brokerURL = strings.TrimLeft(brokerURL, "https://")
	brokerURL = strings.TrimLeft(brokerURL, "http://")

	responseLabels := make(map[string]any, 0)
	responseLabels["Name"] = instance.Parameters.Parameters.Name
	if !IsOwnClusterPlan(instance.ServicePlanID) && instance.RuntimeID != "" {
		responseLabels[kubeconfigURLKey] = fmt.Sprintf("https://%s/kubeconfig/%s", brokerURL, instance.InstanceID)
		apiServerUrl, err := kubeconfigBuilder.GetServerURL(instance.RuntimeID)
		switch {
		case err == nil && apiServerUrl != "":
			responseLabels[apiServerURLKey] = apiServerUrl
		case kubeconfig.IsNotFound(err):
			slog.Info(fmt.Sprintf(apiServerURLErrorFormat, err))
		default:
			slog.Error(fmt.Sprintf(apiServerURLErrorFormat, err))
		}
	}

	return responseLabels
}

func ResponseLabelsWithExpirationInfo(
	instance internal.Instance,
	brokerURL string,
	docsURL string,
	docsKey string,
	expireDuration time.Duration,
	expiryDetailsKey string,
	expiredInfoFormat string,
	kubeconfigBuilder kubeconfig.KcBuilder,
) map[string]any {
	labels := ResponseLabels(instance, brokerURL, kubeconfigBuilder)

	expireTime := instance.CreatedAt.Add(expireDuration)
	hoursLeft := calculateHoursLeft(expireTime)
	if instance.IsExpired() {
		delete(labels, kubeconfigURLKey)
		delete(labels, apiServerURLKey)
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
