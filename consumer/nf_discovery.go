// Copyright 2019 free5GC.org
//
// SPDX-License-Identifier: Apache-2.0
//

package consumer

import (
	"context"
	"fmt"
	"net/http"

	ausf_context "github.com/omec-project/ausf/context"
	"github.com/omec-project/ausf/logger"
	nrf_cache "github.com/omec-project/nrf/nrfcache"
	"github.com/omec-project/openapi/Nnrf_NFDiscovery"
	"github.com/omec-project/openapi/models"
)

/*
func SendSearchNFInstances(nrfUri string, targetNfType, requestNfType models.NfType,
	param Nnrf_NFDiscovery.SearchNFInstancesParamOpts) (*models.SearchResult, error) {
	configuration := Nnrf_NFDiscovery.NewConfiguration()
	configuration.SetBasePath(nrfUri)
	client := Nnrf_NFDiscovery.NewAPIClient(configuration)

	result, rsp, rspErr := client.NFInstancesStoreApi.SearchNFInstances(context.TODO(),
		targetNfType, requestNfType, &param)
	if rspErr != nil {
		return nil, fmt.Errorf("NFInstancesStoreApi Response error: %w", rspErr)
	}
	defer func() {
		if rspCloseErr := rsp.Body.Close(); rspCloseErr != nil {
			logger.ConsumerLog.Errorf("NFInstancesStoreApi Response cannot close: %v", rspCloseErr)
		}
	}()
	if rsp != nil && rsp.StatusCode == http.StatusTemporaryRedirect {
		return nil, fmt.Errorf("Temporary Redirect For Non NRF Consumer")
	}
	return &result, nil
}*/

func SendSearchNFInstances(nrfUri string, targetNfType, requestNfType models.NfType,
	param *Nnrf_NFDiscovery.SearchNFInstancesParamOpts) (models.SearchResult, error) {
	if ausf_context.GetSelf().EnableNrfCaching {
		return nrf_cache.SearchNFInstances(nrfUri, targetNfType, requestNfType, param)
	} else {
		return SendNfDiscoveryToNrf(nrfUri, targetNfType, requestNfType, param)
	}
}

func SendNfDiscoveryToNrf(nrfUri string, targetNfType, requestNfType models.NfType,
	param *Nnrf_NFDiscovery.SearchNFInstancesParamOpts) (models.SearchResult, error) {
	// Set client and set url
	configuration := Nnrf_NFDiscovery.NewConfiguration()
	configuration.SetBasePath(nrfUri)
	client := Nnrf_NFDiscovery.NewAPIClient(configuration)

	logger.ConsumerLog.Infof("Inside SendNFDiscoveryToNrf")
	result, res, err := client.NFInstancesStoreApi.SearchNFInstances(context.TODO(), targetNfType, requestNfType, param)
	if res != nil && res.StatusCode == http.StatusTemporaryRedirect {
		err = fmt.Errorf("Temporary Redirect For Non NRF Consumer")
	}
	defer func() {
		if bodyCloseErr := res.Body.Close(); bodyCloseErr != nil {
			err = fmt.Errorf("SearchNFInstances' response body cannot close: %+w", bodyCloseErr)
		}
	}()

	ausfSelf := ausf_context.GetSelf()

	for _, nfProfile := range result.NfInstances {
		// checking whether the AUSF subscribed to this target nfinstanceid or not
		logger.ConsumerLog.Infof("Rcvd data from NRF %s", nfProfile.NfInstanceId)
		if _, ok := ausfSelf.NfStatusSubscriptions.Load(nfProfile.NfInstanceId); !ok {
			nrfSubscriptionData := models.NrfSubscriptionData{
				NfStatusNotificationUri: fmt.Sprintf("%s/nausf-callback/v1/nf-status-notify", ausfSelf.GetIPv4Uri()),
				SubscrCond:              &models.NfInstanceIdCond{NfInstanceId: nfProfile.NfInstanceId},
				ReqNfType:               requestNfType,
			}
			nrfSubData, problemDetails, err := SendCreateSubscription(nrfUri, nrfSubscriptionData)
			if problemDetails != nil {
				logger.ConsumerLog.Errorf("SendCreateSubscription to NRF, Problem[%+v]", problemDetails)
			} else if err != nil {
				logger.ConsumerLog.Errorf("SendCreateSubscription Error[%+v]", err)
			}
			logger.ConsumerLog.Infof("Storing subscription %s %s %s", ausfSelf.GetIPv4Uri(), nfProfile.NfInstanceId, nrfSubData.SubscriptionId)
			ausfSelf.NfStatusSubscriptions.Store(nfProfile.NfInstanceId, nrfSubData.SubscriptionId)
		}
	}

	return result, err
}
