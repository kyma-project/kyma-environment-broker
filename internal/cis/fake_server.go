package cis

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
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
	eventsEndpoint      *eventsEndpoint
}

type subaccountsEndpoint struct {
	subaccounts map[string]map[string]interface{}
}

type eventsEndpoint struct {
	events []map[string]interface{}
}

type mutableEvents []map[string]interface{}

type eventsEndpointResponse struct {
	Total      int           `json:"total"`
	TotalPages int           `json:"totalPages"`
	PageNum    int           `json:"pageNum"`
	MorePages  bool          `json:"morePages"`
	Events     mutableEvents `json:"events"`
}

func NewFakeServer() (*fakeServer, error) {
	se, err := newSubaccountsEndpoint()
	if err != nil {
		return nil, fmt.Errorf("while creating new subaccounts endpoint: %w", err)
	}
	ee, err := newEventsEndpoint()
	if err != nil {
		return nil, fmt.Errorf("while creating new events endpoint: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /accounts/v1/technical/subaccounts/{subaccountID}", se.getSubaccount)
	mux.HandleFunc("GET /events/v1/events/central", ee.getEvents)

	srv := httptest.NewServer(mux)

	return &fakeServer{
		Server:              srv,
		subaccountsEndpoint: se,
		eventsEndpoint:      ee,
	}, nil
}

func newSubaccountsEndpoint() (*subaccountsEndpoint, error) {
	endpoint := &subaccountsEndpoint{subaccounts: make(map[string]map[string]interface{}, 0)}

	f, err := os.Open(subaccountsJSONPath)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("while reading subaccounts JSON file: %w", err)
	}

	type jsonObjects []map[string]interface{}

	var temp jsonObjects
	d := json.NewDecoder(f)
	err = d.Decode(&temp)
	if err != nil {
		return nil, fmt.Errorf("while decoding subaccounts JSON: %w", err)
	}

	for _, saData := range temp {
		ival, ok := saData[subaccountIDJSONKey]
		if !ok {
			return nil, fmt.Errorf("subaccounts JSON file is invalid - one of objects missing %s key", subaccountIDJSONKey)
		}
		subaccountID, ok := ival.(string)
		if !ok {
			return nil, fmt.Errorf("subaccounts JSON file is invalid - in one of objects value for %s is not a string", subaccountIDJSONKey)
		}
		endpoint.subaccounts[subaccountID] = saData
	}

	return endpoint, nil
}

func (e *subaccountsEndpoint) getSubaccount(w http.ResponseWriter, r *http.Request) {
	subaccountID := r.PathValue("subaccountID")
	if len(subaccountID) == 0 {
		w.WriteHeader(http.StatusBadRequest)
	}
	_, found := e.subaccounts[subaccountID]
	if !found {
		w.WriteHeader(http.StatusNotFound)
	}

	data, err := json.Marshal(e.subaccounts[subaccountID])
	if err != nil {
		log.Println("error while marshalling subaccount data: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Println("error while writing subaccount data: %w", err)
	}
}

func newEventsEndpoint() (*eventsEndpoint, error) {
	endpoint := &eventsEndpoint{events: make([]map[string]interface{}, 0)}

	f, err := os.Open(eventsJSONPath)
	defer f.Close()
	if err != nil {
		return nil, fmt.Errorf("while reading events JSON file: %w", err)
	}

	d := json.NewDecoder(f)
	err = d.Decode(&endpoint.events)
	if err != nil {
		return nil, fmt.Errorf("while decoding events JSON: %w", err)
	}

	return endpoint, nil
}

func (e *eventsEndpoint) getEvents(w http.ResponseWriter, r *http.Request) {
	events := make(mutableEvents, 0, len(e.events))
	events = append(events, e.events...)
	pageSize, _ := strconv.Atoi(defaultPageSize)
	pageNumber := 0

	query := r.URL.Query()
	eventTypeFilter := query.Get("eventType")
	actionTimeFilter := query.Get("fromActionTime")
	sortField := query.Get("sortField")
	sortOrder := strings.ToUpper(query.Get("sortOrder"))
	pageSizeLimit := query.Get("pageSize")
	pageNumberRequest := query.Get("pageNum")

	if eventTypeFilter != "" {
		events.filterEventsByEventType(eventTypeFilter)
	}
	if actionTimeFilter != "" {
		events.filterEventsByActionTime(actionTimeFilter)
	}
	if sortOrder == "" || (sortOrder != "ASC" && sortOrder != "DESC") {
		sortOrder = "ASC"
	}
	if sortField != "" {
		events.sortEvents(sortField, sortOrder)
	}
	if pageSizeLimit != "" {
		sizeLimit, err := strconv.Atoi(pageSizeLimit)
		if err == nil && sizeLimit > 1 {
			pageSize = sizeLimit
		}
	}
	if pageNumberRequest != "" {
		requestedPageNumber, err := strconv.Atoi(pageNumberRequest)
		if err == nil && requestedPageNumber >= 0 {
			pageNumber = requestedPageNumber
		}
	}

	eventsNumber := len(events)
	pagesNumber := int(math.Ceil(float64(eventsNumber) / float64(pageSize)))

	eventsForResponse := make([]map[string]interface{}, 0)
	if len(events) < pageSize {
		eventsForResponse = append(eventsForResponse, events...)
	} else {
		startIndex := pageNumber * pageSize
		endIndex := startIndex + pageSize
		if endIndex > eventsNumber {
			endIndex = eventsNumber
		}
		eventsForResponse = append(eventsForResponse, events[startIndex:endIndex]...)
	}

	resp := eventsEndpointResponse{
		Total:      eventsNumber,
		TotalPages: pagesNumber,
		PageNum:    pageNumber,
		MorePages:  pageNumber < pagesNumber-1,
		Events:     eventsForResponse,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Fatal(fmt.Errorf("while marshalling events data: %w", err))
	}

	w.WriteHeader(http.StatusOK)
	if _, err = w.Write(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (e *mutableEvents) filterEventsByEventType(eventTypeFilter string) {
	for i := 0; i < len(*e); {
		currentEvent := (*e)[i]
		ival, ok := currentEvent[eventTypeJSONKey]
		if !ok {
			log.Println("missing eventType key in one of events")
			break
		}
		actualEventType, ok := ival.(string)
		if !ok {
			log.Println("cannot cast eventType value to string - wrong value in one of events")
			break
		}
		if actualEventType != eventTypeFilter {
			*e = append((*e)[:i], (*e)[i+1:]...)
			continue
		}
		i++
	}
}

func (e *mutableEvents) filterEventsByActionTime(actionTimeFilter string) {
	filterInUnixMilli, err := strconv.ParseInt(actionTimeFilter, 10, 64)
	if err != nil {
		log.Println("cannot parse actionTime filter to int64")
		return
	}

	timeFilter := time.UnixMilli(filterInUnixMilli)
	for i := 0; i < len(*e); {
		currentEvent := (*e)[i]
		ival, ok := currentEvent[actionTimeJSONKey]
		if !ok {
			log.Println("missing actionTime key in one of events")
			break
		}
		eventActionTimeFloat, ok := ival.(float64)
		if !ok {
			log.Println("cannot cast actionTime value to int64 - wrong value in one of events")
			break
		}
		eventActionTimeInUnixMilli := int64(eventActionTimeFloat)
		actualActionTime := time.UnixMilli(eventActionTimeInUnixMilli)
		if actualActionTime.Before(timeFilter) {
			*e = append((*e)[:i], (*e)[i+1:]...)
			continue
		}
		i++
	}
}

func (e *mutableEvents) sortEvents(sortField, sortOrder string) {
	switch sortField {
	case "actionTime":
		e.sortEventsByActionTime(sortOrder)
	default:
		log.Println("unsupported sort field")
	}
}

func (e *mutableEvents) sortEventsByActionTime(sortOrder string) {
	for i := 0; i < len(*e); i++ {
		for j := i + 1; j < len(*e); j++ {
			ival1, ok := (*e)[i][actionTimeJSONKey]
			if !ok {
				log.Println("missing actionTime key in one of events")
				continue
			}
			actionTime1, ok := ival1.(int64)
			if !ok {
				log.Println("cannot cast actionTime value to int64 - wrong value in one of events")
				continue
			}

			ival2, ok := (*e)[j][actionTimeJSONKey]
			if !ok {
				log.Println("missing actionTime key in one of events")
				continue
			}
			actionTime2, ok := ival2.(int64)
			if !ok {
				log.Println("cannot cast actionTime value to int64 - wrong value in one of events")
				continue
			}

			switch sortOrder {
			case "ASC":
				if actionTime1 > actionTime2 {
					(*e)[i], (*e)[j] = (*e)[j], (*e)[i]
				}
			case "DESC":
				if actionTime1 < actionTime2 {
					(*e)[i], (*e)[j] = (*e)[j], (*e)[i]
				}
			}
		}
	}
}
