// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package producer

import (
	"net/http"
	"strings"

	"github.com/omec-project/ausf/logger"

	ausf_context "github.com/omec-project/ausf/context"

	"github.com/omec-project/ausf/consumer"

	nrf_cache "github.com/omec-project/nrf/nrfcache"

	"github.com/omec-project/http_wrapper"
	"github.com/omec-project/openapi/models"
)

func HandleNfSubscriptionStatusNotify(request *http_wrapper.Request) *http_wrapper.Response {
	logger.ProducerLog.Traceln("[AUSF] Handle NF Status Notify")

	notificationData := request.Body.(models.NotificationData)

	problemDetails := NfSubscriptionStatusNotifyProcedure(notificationData)
	if problemDetails != nil {
		return http_wrapper.NewResponse(int(problemDetails.Status), nil, problemDetails)
	} else {
		return http_wrapper.NewResponse(http.StatusNoContent, nil, nil)
	}
}

func NfSubscriptionStatusNotifyProcedure(notificationData models.NotificationData) *models.ProblemDetails {
	logger.ProducerLog.Debugf("NfSubscriptionStatusNotify: %+v", notificationData)

	if notificationData.Event == "" || notificationData.NfInstanceUri == "" {
		problemDetails := &models.ProblemDetails{
			Status: http.StatusBadRequest,
			Cause:  "MANDATORY_IE_MISSING", // Defined in TS 29.510 6.1.6.2.17
			Detail: "Missing IE [Event]/[NfInstanceUri] in NotificationData",
		}
		return problemDetails
	}
	nfInstanceId := notificationData.NfInstanceUri[strings.LastIndex(notificationData.NfInstanceUri, "/")+1:]

	logger.ProducerLog.Infof("Recieved Subscription Status Notification from NRF: %v", notificationData.Event)
	// If nrf caching is enabled, go ahead and delete the entry from the cache.
	// This will force the ausf to do nf discovery and get the updated nf profile from the nrf.
	if notificationData.Event == models.NotificationEventType_DEREGISTERED {
		if ausf_context.GetSelf().EnableNrfCaching {
			ok := nrf_cache.RemoveNfProfileFromNrfCache(nfInstanceId)
			logger.ProducerLog.Tracef("nfinstance %v deleted from cache: %v", nfInstanceId, ok)
		}
		if subscriptionId, ok := ausf_context.GetSelf().NfStatusSubscriptions.Load(nfInstanceId); ok {
			logger.ConsumerLog.Debugf("SubscriptionId of nfInstance %v is %v", nfInstanceId, subscriptionId.(string))
			problemDetails, err := consumer.SendRemoveSubscription(subscriptionId.(string))
			if problemDetails != nil {
				logger.ConsumerLog.Errorf("Remove NF Subscription Failed Problem[%+v]", problemDetails)
			} else if err != nil {
				logger.ConsumerLog.Errorf("Remove NF Subscription Error[%+v]", err)
			} else {
				logger.ConsumerLog.Infoln("[AUSF] Remove NF Subscription successful")
				ausf_context.GetSelf().NfStatusSubscriptions.Delete(nfInstanceId)
			}
		} else {
			logger.ProducerLog.Infof("nfinstance %v not found in map", nfInstanceId)
		}
	}

	return nil
}
