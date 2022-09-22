// Package dnyndns the libdns interfaces for dnyndns.Customize all godocs for actual implementation.
package dnyndns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/libdns/libdns"
)

const (
	apiUrl = "https://api.dynu.com/v2"
)

type DnsRecordResponse struct {
	DnsRecords []DnsRecord `json:"dnsRecords"`
}

type DnsRecord struct {
	Id              int    `json:"id"`
	DomainId        int    `json:"domainId"`
	DomainName      string `json:"domainName"`
	NodeName        string `json:"nodeName"`
	Hostname        string `json:"hostname"`
	RecordType      string `json:"recordType"`
	Ttl             int    `json:"ttl"`
	State           int    `json:"state"`
	Content         string `json:"content"`
	UpdatedOn       string `json:"updatedOn"`
	MasterName      string `json:"masterName"`
	ResponsibleName string `json:"responsibleName"`
	Refresh         int    `json:"refresh"`
	Retry           int    `json:"retry"`
	Expire          int    `json:"expire"`
	NegativeTTL     int    `json:"negativeTTL"`
	Ipv4Address     string `json:"ipv4Address"`
	TextData        string `json:"textData"`
	Group           string `json:"group"`
}

type DomainResponse struct {
	Id         int    `json:"id"`
	DomainName string `json:"domainName"`
	Hostname   string `json:"hostname"`
	Node       string `json:"node"`
}

func getDomain(zone string) string {
	return strings.TrimSuffix(zone, ".")
}

func getDomainIdFromFQDN(apiKey string, ResolvedFQDN string) (string, string, error) {
	hostname := getDomain(ResolvedFQDN)
	url := apiUrl + "/dns/getroot/" + hostname
	response, err := callDnsApi(url, "GET", nil, apiKey)
	if err != nil {
		return "", "", err
	}

	domainResponse := DomainResponse{}
	readErr := json.Unmarshal(response, &domainResponse)
	if readErr != nil {
		return "", "", readErr
	}
	return fmt.Sprint(domainResponse.Id), domainResponse.Node, nil
}

func getRecordsForDomain(apiKey string, domainId string) ([]byte, error) {
	url := apiUrl + "/dns/" + domainId + "/record"
	response, err := callDnsApi(url, "GET", nil, apiKey)

	return response, err
}

func getRecordsForDomainByZone(apiKey string, zone string) (DnsRecordResponse, string, error) {
	dnsRecordsResponse := DnsRecordResponse{}
	domainId, _, err := getDomainIdFromFQDN(apiKey, zone)
	if err != nil {
		return dnsRecordsResponse, "", err
	}
	dnsRecords, err := getRecordsForDomain(apiKey, domainId)
	if err != nil {
		return dnsRecordsResponse, "", fmt.Errorf("unable to get DNS records %v", err)
	}
	readErr := json.Unmarshal(dnsRecords, &dnsRecordsResponse)
	return dnsRecordsResponse, domainId, readErr
}

func getRecodsMaps(apiKey string, zone string) (map[string]map[string]int, string, error) {
	dnsRecordsResponse, domainId, err := getRecordsForDomainByZone(apiKey, zone)
	recordIDs := make(map[string]map[string]int)
	for i := len(dnsRecordsResponse.DnsRecords) - 1; i >= 0; i-- {
		record := dnsRecordsResponse.DnsRecords[i]

		if _, exist := recordIDs[record.RecordType]; exist {
			recordIDs[record.RecordType][record.NodeName] = record.Id
		} else {
			c := make(map[string]int)
			c[record.NodeName] = record.Id
			recordIDs[record.RecordType] = c
		}
	}
	return recordIDs, domainId, err
}

func deleteRecord(apiKey string, domainId string, recordId int) (string, error) {
	url := apiUrl + "/dns/" + domainId + "/record/" + fmt.Sprint(recordId)
	response, err := callDnsApi(url, "DELETE", nil, apiKey)
	return string(response), err
}

func getRecordReqBody(recordName string, recordType string, value string, ttl time.Duration) ([]byte, error) {
	requestbody := map[string]string{
		"nodeName":   recordName,
		"recordType": recordType,
		"ttl":        fmt.Sprint(ttl * time.Second),
		"group":      "",
		"state":      "true"}
	if recordType == "TXT" {
		requestbody["textData"] = value
	} else if recordType == "A" {
		requestbody["ipv4Address"] = value
	}
	jsonBody, err := json.Marshal(requestbody)
	return jsonBody, err
}

func addRecord(apiKey string, domainId string, recordName string, recordType string, value string, ttl time.Duration) {
	jsonBody, err := getRecordReqBody(recordName, recordType, value, ttl)
	if err != nil {
		log.Println(err)
		return
	}
	url := apiUrl + "/dns/" + domainId + "/record"
	response, err := callDnsApi(url, "POST", bytes.NewBuffer(jsonBody), apiKey)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Printf("Add  record result: %s", string(response))
}

func updteRecord(apiKey string, domainId string, recordId int, recordName string, recordType string, value string, ttl time.Duration) {
	jsonBody, err := getRecordReqBody(recordName, recordType, value, ttl)
	if err != nil {
		log.Println(err)
		return
	}
	url := apiUrl + "/dns/" + domainId + "/record/" + fmt.Sprint(recordId)
	response, err := callDnsApi(url, "POST", bytes.NewBuffer(jsonBody), apiKey)

	if err != nil {
		log.Println(err)
	}
	fmt.Printf("updteRecord record result: %s", string(response))
}

func callDnsApi(url string, method string, body io.Reader, apiKey string) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return []byte{}, fmt.Errorf("unable to execute request %v", err)
	}
	req.Close = true
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("API-Key", apiKey)
	t := &http.Transport{
		TLSHandshakeTimeout: 60 * time.Second,
	}
	client := &http.Client{
		Transport: t,
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Failed to Do request")
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if resp.StatusCode == http.StatusOK {
		return respBody, nil
	}

	text := "Error calling API status:" + resp.Status + " url: " + url + " method: " + method
	log.Println(text)
	return nil, errors.New(text)
}

type Provider struct {
	// struct tags on exported fields), for example:
	APIToken string `json:"api_token,omitempty"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	dnsRecordsResponse, _, err := getRecordsForDomainByZone(p.APIToken, zone)
	if err != nil {
		return nil, fmt.Errorf("unable to get DNS records %v", err)
	}
	var records []libdns.Record
	for i := len(dnsRecordsResponse.DnsRecords) - 1; i >= 0; i-- {
		record := dnsRecordsResponse.DnsRecords[i]
		val := ""
		if record.RecordType == "TXT" {
			val = record.TextData
		} else if record.RecordType == "A" {
			val = record.Ipv4Address
		}
		records = append(records, libdns.Record{
			Type:  record.RecordType,
			Name:  record.NodeName,
			Value: val,
			TTL:   time.Duration(record.Ttl) * time.Second,
		})
	}
	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("AppendRecords", zone, records)
	var appendedRecords []libdns.Record
	recordIDs, domainId, err := getRecodsMaps(p.APIToken, zone)
	if err != nil {
		return nil, fmt.Errorf("unable to get DNS records %v", err)
	}

	for _, record := range records {
		_, exist := recordIDs[record.Type]
		recordId := 0
		if exist {
			recordId = recordIDs[record.Type][record.Name]
		}
		if recordId != 0 {
			updteRecord(p.APIToken, domainId, recordId, record.Name, record.Type, record.Value, record.TTL)
		} else {
			addRecord(p.APIToken, domainId, record.Name, record.Type, record.Value, record.TTL)
		}
		if err != nil {
			return nil, err
		}
		appendedRecords = append(appendedRecords, record)
	}

	return appendedRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	return p.AppendRecords(ctx, zone, records)
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	log.Println("DeleteRecords", zone, records)
	var deleteRecords []libdns.Record

	recordIDs, domainId, err := getRecodsMaps(p.APIToken, zone)
	if err != nil {
		return nil, fmt.Errorf("unable to get DNS records %v", err)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to get DNS records %v", err)
	}

	for _, record := range records {
		_, exist := recordIDs[record.Type]
		if !exist {
			return nil, fmt.Errorf("unable to get DNS records %v", err)
		}
		recordId, exist1 := recordIDs[record.Type][record.Name]
		if !exist1 {
			return nil, fmt.Errorf("unable to get DNS records %v", err)
		}
		_, err := deleteRecord(p.APIToken, domainId, recordId)
		if err != nil {
			return nil, err
		}
		deleteRecords = append(deleteRecords, record)
	}
	return deleteRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
