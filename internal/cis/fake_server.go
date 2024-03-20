package cis

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
)

const (
	FakeSubaccountID1   = "cad2806a-3545-4aa0-8a7c-4fc246dba684"
	FakeSubaccountID2   = "17b8dcc2-3de1-4884-bcd3-b1c4657d81be"
	subaccountsJSONPath = "testdata/subaccounts.json"
	subaccountIDJSONKey = "guid"
)

type fakeServer struct {
	*httptest.Server
	subaccountsEndpoint *subaccountsEndpoint
}

type subaccountsEndpoint struct {
	subaccounts map[string]map[string]interface{}
}

func NewFakeServer() *fakeServer {
	se := newSubaccountsEndpoint()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/v1/technical/subaccounts/{subaccountID}", se.getSubaccount)

	srv := httptest.NewServer(mux)

	return &fakeServer{
		Server:              srv,
		subaccountsEndpoint: se,
	}
}

func newSubaccountsEndpoint() *subaccountsEndpoint {
	endpoint := &subaccountsEndpoint{subaccounts: make(map[string]map[string]interface{}, 0)}

	f, err := os.Open(subaccountsJSONPath)
	defer f.Close()
	if err != nil {
		log.Fatal(fmt.Errorf("while reading subaccounts JSON file: %w", err))
	}

	type jsonObjects []map[string]interface{}

	var temp jsonObjects
	d := json.NewDecoder(f)
	err = d.Decode(&temp)
	if err != nil {
		log.Fatal(fmt.Errorf("while decoding subaccounts JSON: %w", err))
	}

	for _, saData := range temp {
		ival, ok := saData[subaccountIDJSONKey]
		if !ok {
			log.Fatal(fmt.Errorf("subaccounts JSON file is invalid - one of objects missing %s key", subaccountIDJSONKey))
		}
		subaccountID, ok := ival.(string)
		if !ok {
			log.Fatal(fmt.Errorf("subaccounts JSON file is invalid - in one of objects value for %s is not a string", subaccountIDJSONKey))
		}
		endpoint.subaccounts[subaccountID] = saData
	}

	return endpoint
}

func (e *subaccountsEndpoint) getSubaccount(w http.ResponseWriter, r *http.Request) {
	subaccountID := r.PathValue("subaccountID")
	if len(subaccountID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	_, found := e.subaccounts[subaccountID]
	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := json.Marshal(e.subaccounts[subaccountID])
	if err != nil {
		log.Fatal(fmt.Errorf("while marshalling subaccount data: %w", err))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
