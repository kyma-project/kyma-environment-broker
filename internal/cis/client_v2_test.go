package cis

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kyma-project/kyma-environment-broker/internal/httputil"

	"github.com/stretchr/testify/require"
)

const (
	subAccountV2Test1 = "fda14cab-bacc-4d0b-a10f-18557a6d9060"
	subAccountV2Test2 = "7514cf27-41b0-4266-a273-637cb3a2c051"
	subAccountV2Test3 = "47af15c8-adfe-4404-8675-525a878c4601"
)

func TestClientV2_FetchSubaccountsToDelete(t *testing.T) {
	t.Run("client fetched all subaccount IDs to delete", func(t *testing.T) {
		// Given
		testServer := fixHTTPServerV2(newServerV2(t))
		defer testServer.Close()

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		client := NewClientV2(context.TODO(), Config{
			EventServiceURL: testServer.URL,
			PageSize:        "3",
		}, logger)
		client.SetHttpClient(testServer.Client())

		// When
		saList, err := client.FetchSubaccountsToDelete()

		// Then
		require.NoError(t, err)
		require.Len(t, saList, 3)
		require.ElementsMatch(t, saList, []string{subAccountV2Test1, subAccountV2Test2, subAccountV2Test3})
	})

	t.Run("error occur during fetch subaccount IDs", func(t *testing.T) {
		// Given
		srv := newServerV2(t)
		srv.serverErr = true
		testServer := fixHTTPServerV2(srv)
		defer testServer.Close()

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		client := NewClientV2(context.TODO(), Config{
			EventServiceURL: testServer.URL,
			PageSize:        "3",
		}, logger)
		client.SetHttpClient(testServer.Client())

		// When
		saList, err := client.FetchSubaccountsToDelete()

		// Then
		require.Error(t, err)
		require.Len(t, saList, 0)
	})

	t.Run("should fetch subaccounts ids after request retries", func(t *testing.T) {
		// Given
		srv := newServerV2(t)
		srv.rateLimiting = true
		srv.requiredRequestRetries = 1
		testServer := fixHTTPServerV2(srv)
		defer testServer.Close()

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		client := NewClientV2(context.TODO(), Config{
			EventServiceURL:   testServer.URL,
			PageSize:          "3",
			MaxRequestRetries: 3,
		}, logger)
		client.SetHttpClient(testServer.Client())

		// When
		saList, err := client.FetchSubaccountsToDelete()

		// Then
		require.NoError(t, err)
		require.Len(t, saList, 3)
		require.ElementsMatch(t, saList, []string{subAccountV2Test1, subAccountV2Test2, subAccountV2Test3})
	})

	t.Run("should return rate limiting error", func(t *testing.T) {
		// Given
		srv := newServerV2(t)
		srv.rateLimiting = true
		srv.requiredRequestRetries = 5
		testServer := fixHTTPServerV2(srv)
		defer testServer.Close()

		logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		client := NewClientV2(context.TODO(), Config{
			EventServiceURL:   testServer.URL,
			PageSize:          "3",
			MaxRequestRetries: 3,
		}, logger)
		client.SetHttpClient(testServer.Client())

		// When
		saList, err := client.FetchSubaccountsToDelete()

		// Then
		require.Error(t, err)
		require.Len(t, saList, 0)
	})
}

type serverV2 struct {
	t                      *testing.T
	serverErr              bool
	rateLimiting           bool
	requestRetriesCount    int
	requiredRequestRetries int
}

func newServerV2(t *testing.T) *serverV2 {
	return &serverV2{
		t: t,
	}
}

func fixHTTPServerV2(srv *serverV2) *httptest.Server {
	r := httputil.NewRouter()

	r.HandleFunc("GET /events/v2/events/central", srv.returnCISEventsV2)

	return httptest.NewServer(r)
}

func (s *serverV2) returnCISEventsV2(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("eventType")
	cursor := r.URL.Query().Get("cursor")

	if eventType != "Subaccount_Deletion" && cursor == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if s.serverErr {
		s.writeResponseV2(w, []byte(`{bad}`))
		return
	}

	if s.rateLimiting {
		if s.requestRetriesCount < s.requiredRequestRetries {
			s.writeRateLimitingResponseV2(w)
			s.requestRetriesCount++
			return
		}
	}

	var response string
	if cursor != "" {
		response = `{"events": []}`
	} else {
		response = fmt.Sprintf(`{
			"nextCursor": "",
			"events": [
				{
					"id": 1001,
					"actionTime": 1597135762286,
					"creationTime": 1597135763081,
					"details": {
						"description": "Subaccount deleted.",
						"guid": "%s",
						"technicalName": "%s",
						"parentGuid": "a6c5f1b0-9713-45fc-a831-ed0057a7925c",
						"displayName": "trial",
						"subaccountDescription": null,
						"region": "eu10",
						"jobLocation": null,
						"subdomain": "trial-subdomain",
						"betaEnabled": false,
						"labels": {},
						"usedForProduction": "UNSET"
					},
					"globalAccountGUID": "a6c5f1b0-9713-45fc-a831-ed0057a7925c",
					"entityId": "%s",
					"entityType": "Subaccount",
					"eventOrigin": "accounts-service",
					"eventType": "Subaccount_Deletion"
				},
				{
					"id": 1002,
					"actionTime": 1597090087820,
					"creationTime": 1597090088405,
					"details": {
						"description": "Subaccount deleted.",
						"guid": "%s",
						"technicalName": "%s",
						"parentGuid": "ec0a066a-60a1-4d31-b329-80cf97292789",
						"displayName": "test-subaccount",
						"subaccountDescription": null,
						"region": "eu10",
						"jobLocation": null,
						"subdomain": "test-subdomain",
						"betaEnabled": false,
						"labels": {},
						"usedForProduction": "UNSET"
					},
					"globalAccountGUID": "ec0a066a-60a1-4d31-b329-80cf97292789",
					"entityId": "%s",
					"entityType": "Subaccount",
					"eventOrigin": "accounts-service",
					"eventType": "Subaccount_Deletion"
				},
				{
					"id": 1003,
					"actionTime": 1597090066116,
					"creationTime": 1597090067309,
					"details": {
						"description": "Subaccount deleted.",
						"guid": "%s",
						"technicalName": "%s",
						"parentGuid": "ec0a066a-60a1-4d31-b329-80cf97292789",
						"displayName": "dev-subaccount",
						"subaccountDescription": null,
						"region": "eu10",
						"jobLocation": null,
						"subdomain": "dev-subdomain",
						"betaEnabled": false,
						"labels": {},
						"usedForProduction": "UNSET"
					},
					"globalAccountGUID": "ec0a066a-60a1-4d31-b329-80cf97292789",
					"entityId": "%s",
					"entityType": "Subaccount",
					"eventOrigin": "accounts-service",
					"eventType": "Subaccount_Deletion"
				}
			]
		}`, subAccountV2Test1, subAccountV2Test1, subAccountV2Test1,
			subAccountV2Test2, subAccountV2Test2, subAccountV2Test2,
			subAccountV2Test3, subAccountV2Test3, subAccountV2Test3)
	}

	s.writeResponseV2(w, []byte(response))
	s.requestRetriesCount = 0
}

func (s *serverV2) writeResponseV2(w http.ResponseWriter, response []byte) {
	_, err := w.Write(response)
	if err != nil {
		s.t.Errorf("fakeCisServerV2 cannot write response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *serverV2) writeRateLimitingResponseV2(w http.ResponseWriter) {
	response := `{
		"error": {
			"message": "Request rate limit exceeded"
		}
	}`
	w.WriteHeader(http.StatusTooManyRequests)
	_, err := w.Write([]byte(response))
	if err != nil {
		s.t.Errorf("fakeCisServerV2 cannot write response: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
