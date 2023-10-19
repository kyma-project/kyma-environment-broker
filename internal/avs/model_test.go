package avs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kyma-project/kyma-environment-broker/internal"
	"github.com/stretchr/testify/assert"
)

var evalIdsHolder []int64
var parentEvalIdHolder map[int64]int64 = make(map[int64]int64)

const (
	internalEvalId        = int64(1234)
	externalEvalId        = int64(5678)
	internalEvalUpdatedId = int64(12340)
	externalEvalUpdatedId = int64(56780)
)

func TestAvsEvaluationConfigs(t *testing.T) {
	// given
	assert := assert.New(t)

	mockOauthServer := newMockAvsOauthServer()
	defer mockOauthServer.Close()
	mockAvsServer := newMockAvsServer(t)
	defer mockAvsServer.Close()
	avsConfig := avsConfig(mockOauthServer, mockAvsServer)
	internalEvalAssistant := NewInternalEvalAssistant(avsConfig)
	externalEvalAssistant := NewExternalEvalAssistant(avsConfig)

	// verify assistant configs
	assert.Equal(internalEvalId, internalEvalAssistant.ProvideTesterAccessId(internal.ProvisioningParameters{}))
	assert.Equal(externalEvalId, externalEvalAssistant.ProvideTesterAccessId(internal.ProvisioningParameters{}))

	assert.Equal("int", internalEvalAssistant.ProvideSuffix())
	assert.Equal("ext", externalEvalAssistant.ProvideSuffix())

	assert.Equal("", internalEvalAssistant.ProvideCheckType())
	assert.Equal("HTTPSGET", externalEvalAssistant.ProvideCheckType())

	assert.Equal("dummy", internalEvalAssistant.ProvideNewOrDefaultServiceName("dummy"))
	assert.Equal("external-dummy", externalEvalAssistant.ProvideNewOrDefaultServiceName("dummy"))

	params := internal.Operation{}
	assert.Equal(7, len(internalEvalAssistant.ProvideTags(internal.Operation{})))

	// verify confg as json
	tags, testTag := externalEvalAssistant.ProvideTags(params), Tag{}
	json.Unmarshal([]byte(`{"content":"","tag_class_id":0,"tag_class_name":""}`), &testTag)
	assert.Equal(testTag, *tags[0])

}

func newMockAvsOauthServer() *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"access_token": "90d64460d14870c08c81352a05dedd3465940a7c", "scope": "user", "token_type": "bearer", "expires_in": 86400}`))
		}))
}

func newMockAvsServer(t *testing.T) *httptest.Server {
	router := mux.NewRouter()
	router.HandleFunc("/{evalId}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		evalIdsHolder = append(evalIdsHolder, extractId(vars, "evalId", t))
		w.WriteHeader(http.StatusOK)
	})).Methods(http.MethodDelete)

	router.HandleFunc("/{parentId}/child/{evalId}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		parentEval := extractId(vars, "parentId", t)
		evalId := extractId(vars, "evalId", t)
		parentEvalIdHolder[evalId] = parentEval

		w.WriteHeader(http.StatusOK)
	})).Methods(http.MethodDelete)
	return httptest.NewServer(router)
}

func extractId(vars map[string]string, key string, t *testing.T) int64 {
	evalIdStr := vars[key]
	evalId, err := strconv.ParseInt(evalIdStr, 10, 64)
	assert.NoError(t, err)
	return evalId
}

func avsConfig(mockOauthServer *httptest.Server, mockAvsServer *httptest.Server) Config {
	return Config{
		OauthTokenEndpoint:     mockOauthServer.URL,
		OauthUsername:          "dummy",
		OauthPassword:          "dummy",
		OauthClientId:          "dummy",
		ApiEndpoint:            mockAvsServer.URL,
		DefinitionType:         DefinitionType,
		InternalTesterAccessId: internalEvalId,
		InternalTesterService:  "",
		ExternalTesterAccessId: externalEvalId,
		ExternalTesterService:  "external-dummy",
		GroupId:                5555,
		ParentId:               91011,
	}
}
