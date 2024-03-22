package cis

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"time"
)

const (
	FakeSubaccountID1   = "cad2806a-3545-4aa0-8a7c-4fc246dba684"
	FakeSubaccountID2   = "17b8dcc2-3de1-4884-bcd3-b1c4657d81be"
	eventsJSONPath      = "testdata/events.json"
	subaccountsJSONPath = "testdata/subaccounts.json"
	subaccountIDJSONKey = "guid"
	eventTypeJSONKey    = "eventType"
	actionTimeJSONKey   = "actionTime"
)

type fakeServer struct {
	*httptest.Server
	subaccountsEndpoint *subaccountsEndpoint
}

type subaccountsEndpoint struct {
	subaccounts map[string]map[string]interface{}
}

type eventsEndpoint struct {
	events []map[string]interface{}
}

type mutableEvents []map[string]interface{}

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

func newEventsEndpoint() *eventsEndpoint {
	endpoint := &eventsEndpoint{events: make([]map[string]interface{}, 0)}

	f, err := os.Open(eventsJSONPath)
	defer f.Close()
	if err != nil {
		log.Fatal(fmt.Errorf("while reading events JSON file: %w", err))
	}

	d := json.NewDecoder(f)
	err = d.Decode(&endpoint.events)
	if err != nil {
		log.Fatal(fmt.Errorf("while decoding events JSON: %w", err))
	}

	return endpoint
}

func (e *eventsEndpoint) getEvents(w http.ResponseWriter, r *http.Request) {
	events := make(mutableEvents, len(e.events))
	events = append(events, e.events...)

	query := r.URL.Query()
	eventTypeFilter := query.Get("eventType")
	actionTimeFilter := query.Get("fromActionTime")

	if eventTypeFilter != "" {
		events.filterEventsByEventType(eventTypeFilter)
	}
	if actionTimeFilter != "" {
		events.filterEventsByActionTime(actionTimeFilter)
	}

	data, err := json.Marshal(events)
	if err != nil {
		log.Fatal(fmt.Errorf("while marshalling events data: %w", err))
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (e *mutableEvents) filterEventsByEventType(eventTypeFilter string) {
	for i, event := range *e {
		ival, ok := event[eventTypeJSONKey]
		if !ok {
			log.Println("missing eventType key in one of events")
			continue
		}
		actualEventType, ok := ival.(string)
		if !ok {
			log.Println("cannot cast eventType value to string - wrong value in one of events")
			continue
		}
		if actualEventType != eventTypeFilter {
			*e = append((*e)[:i], (*e)[i+1:]...)
		}
	}
}

func (e *mutableEvents) filterEventsByActionTime(actionTimeFilter string) {
	filterInUnixMilli, err := strconv.ParseInt(actionTimeFilter, 10, 64)
	if err != nil {
		log.Println("cannot parse actionTime filter to int64")
		return
	}

	timeFilter := time.UnixMilli(filterInUnixMilli)
	for i, event := range *e {
		ival, ok := event[actionTimeJSONKey]
		if !ok {
			log.Println("missing actionTime key in one of events")
			continue
		}
		actualActionTimeInUnixMilli, ok := ival.(int64)
		if !ok {
			log.Println("cannot cast actionTime value to int64 - wrong value in one of events")
			continue
		}
		actualActionTime := time.UnixMilli(actualActionTimeInUnixMilli)
		if actualActionTime.Before(timeFilter) {
			*e = append((*e)[:i], (*e)[i+1:]...)
		}
	}
}
