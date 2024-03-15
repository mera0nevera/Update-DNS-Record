package FixDNS

import (
    "bytes"
    "crypto/tls"
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "net/http"
    "strconv"
    "strings"
)

type Powerdns struct {
    Hostname                    string
    Apikey                      string
    VerifySSL                   bool
    BaseURL                     string
    NameServers                 []string
    client                      *http.Client
}

func NewPowerdns(HostName string, ApiKey string, NameServers []string) *Powerdns {
    var powerdns *Powerdns
    var tr *http.Transport

    powerdns = new(Powerdns)
    powerdns.Hostname = HostName
    powerdns.Apikey = ApiKey
    powerdns.VerifySSL = false
    powerdns.BaseURL = "http://" + powerdns.Hostname + ":8081/api/v1/servers/localhost/"
    powerdns.NameServers = NameServers
    if powerdns.VerifySSL {
        tr = &http.Transport{}
    } else {
        tr = &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }
    }
    powerdns.client = &http.Client{Transport: tr}
    return powerdns
}

//Search the data inside PowerDNS for search_term and return at most max_results
func (FixDNS *Powerdns) Search(endpoint string) (interface{}, error) {
    var target string
    var data interface{}
    fmt.Println("PDNS.Get: endpoint: " + endpoint)
    target = FixDNS.BaseURL + endpoint
    req, err := http.NewRequest("GET", target, nil)
    req.Header.Add("Content-Type", "application/json")
    req.Header.Add("Accept", "application/json")
    req.Header.Add("X-API-Key", FixDNS.Apikey)
    r, err := FixDNS.client.Do(req)
    defer r.Body.Close()
    if err != nil {
        fmt.Println("Error while getting")
        fmt.Println(err)
        return nil, err
    }
    if r.StatusCode < 200 || r.StatusCode > 299 {
        return nil, errors.New("HTTP Error " + r.Status)
    }

    response, err := ioutil.ReadAll(r.Body)
    if err != nil {
        fmt.Println("Error while reading body")
        fmt.Println(err)
        return nil, err
    }
    err = json.Unmarshal(response, &data)
    if err != nil {
        fmt.Println("Error while processing JSON")
        fmt.Println(err)
        return nil, err
    }
    return data, nil
}

//Updating soa_edit_api and soa_edit at path zones
func (FixDNS *Powerdns) GetTopDomain(domain string) (topdomain string, err error) {
    fmt.Println("PDNS.GetTopDomain: Domain: " + domain)
    topSlice := strings.Split(domain, ".")
    for i := 0; i < len(topSlice); i++ {
        topdomain = ""
        for n := i; n < len(topSlice); n++ {
            topdomain = topdomain + topSlice[n] + "."
        }
        topDomainData, err := FixDNS.Search("zones/" + topdomain)
        if err == nil {
            topDomainDataMap := topDomainData.(map[string]interface{})
            if topDomainDataMap["soa_edit_api"] != "INCEPTION-INCREMENT" {
                fmt.Println("PDNS.GetTopDomain: Updating soa_edit_api and soa_edit at path zones/" + topdomain)
                update := make(map[string]string)
                update["soa_edit_api"] = "INCEPTION-INCREMENT"
                update["soa_edit"] = "INCEPTION-INCREMENT"
                update["kind"] = "Master"
                jsonText, err := json.Marshal(update)
                err = FixDNS.Put("zones/"+topdomain, jsonText)
                if err != nil {
                    fmt.Println("PDNS.GetTopDomain: Error updating soa_edit_api and soa_edit at path zones/" + topdomain + " ,content:" + string(jsonText))
                    fmt.Println(err)
                }
            }
            return topdomain, err
        }
    }
    return topdomain, errors.New("PDNS.GetTopDomain: Did not found domain:" + domain + " for topdomain:" + topdomain)
}

//Creating new RRset
func (FixDNS *Powerdns) UpdateRec(domain string, dtype string, name string, content string, ttl int) error {

    var recordSlice []interface{}
    var rrSlice []interface{}
    fmt.Println("PDNS.UpdateRec: Domain: " + domain + " ,dtype:" + dtype + " ,name:" + name + " ,content:" + content + " ,ttl:" + strconv.Itoa(ttl))
    record := Record{
        Content:  content,
        Disabled: false,
        Name:     name,
        TTL:      ttl,
        DType:    dtype,
    }
    recordSlice = append(recordSlice, record)
    rrSet := RrSet{
        Name:       name,
        DType:      dtype,
        TTL:        ttl,
        ChangeType: "REPLACE",
        Records:    recordSlice,
    }
    rrSlice = append(rrSlice, rrSet)
    update := make(map[string]interface{})
    update["rrsets"] = rrSlice
    jsonText, err := json.Marshal(update)
    topDomain, err := FixDNS.GetTopDomain(domain)
    if err != nil {
        fmt.Println("PDNS.UpdateRec: Could not get topdomain, reverting to domain: " + domain)
        fmt.Println(err)
        topDomain = domain
    }
    err = FixDNS.Patch("zones/"+topDomain, jsonText)
    if err != nil {
        fmt.Println("PDNS.UpdateRec: Error updating record at path zones/" + topDomain + " ,content:" + string(jsonText))
        fmt.Println(err)
        return err
    }
    return nil
}

//Modifies basic zone data
func (FixDNS *Powerdns) Put(endpoint string, jsonData []byte) (err error) {
    var target string
    var req *http.Request

    fmt.Println("PDNS.Put: endpoint: " + endpoint + " ,jsonData:" + string(jsonData))
    target = FixDNS.BaseURL + endpoint
    //fmt.Println("POST form " + target)
    req, err = http.NewRequest("PUT", target, bytes.NewBuffer(jsonData))
    req.Header.Add("Content-Type", "application/json")
    req.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
    req.Header.Add("Accept", "application/json")
    req.Header.Add("X-API-Key", FixDNS.Apikey)
    r, err := FixDNS.client.Do(req)
    defer r.Body.Close()
    if err != nil {
        fmt.Println("Error while patching")
        fmt.Println(err)
        return err
    }
    if r.StatusCode < 200 || r.StatusCode > 299 {
        return errors.New("HTTP Error " + r.Status)
    }
    return nil
}

//Creates/modifies/deletes RRsets present in the payload and their comments
func (FixDNS *Powerdns) Patch(endpoint string, jsonData []byte) (err error) {
    var target string
    var req *http.Request
    fmt.Println("PDNS.Patch: endpoint: " + endpoint + " ,jsonData:" + string(jsonData))
    target = FixDNS.BaseURL + endpoint
    //fmt.Println("POST form " + target)
    req, err = http.NewRequest("PATCH", target, bytes.NewBuffer(jsonData))
    req.Header.Add("Content-Type", "application/json")
    req.Header.Add("Content-Length", strconv.Itoa(len(jsonData)))
    req.Header.Add("Accept", "application/json")
    req.Header.Add("X-API-Key", FixDNS.Apikey)
    r, err := FixDNS.client.Do(req)
    defer r.Body.Close()
    if err != nil {
        fmt.Println("Error while patching")
        fmt.Println(err)
        return err
    }
    if r.StatusCode < 200 || r.StatusCode > 299 {
        return errors.New("HTTP Error " + r.Status)
    }
    return nil
}

type RrSet struct {
    Name       string        `json:"name"`
    DType      string        `json:"type"`
    TTL        int           `json:"ttl"`
    ChangeType string        `json:"changetype"`
    Records    []interface{} `json:"records"`
}

type Record struct {
    Content  string `json:"content"`
    Disabled bool   `json:"disabled"`
    Name     string `json:"name"`
    TTL      int    `json:"ttl"`
    DType    string `json:"type"`
}

//function for simplify rrset update
func (FixDNS *Powerdns) PushDNSRecordHostPtr(record map[string]interface{}) {
    // fmt.Println(record["name"].(string), record["content"].(string))
    ipAddressSlice := strings.Split(record["content"].(string), ".")
    ptrRecord := ipAddressSlice[3] + "." + ipAddressSlice[2] + "." + ipAddressSlice[1] + "." + ipAddressSlice[0] + ".in-addr.arpa."
    ptrDomain := ipAddressSlice[2] + "." + ipAddressSlice[1] + "." + ipAddressSlice[0] + ".in-addr.arpa"
    hostFqdn := record["name"].(string)
    err := FixDNS.UpdateRec(ptrDomain, "PTR", ptrRecord, hostFqdn, 10)
    if err != nil {
        fmt.Println("Failed to update PTR record, domain: " + ptrDomain + ", content: " + ptrRecord + ", value: " + hostFqdn + " !")
        return
    }
    fmt.Println("Updated PTR record, domain: " + ptrDomain + ", content: " + ptrRecord + ", value: " + hostFqdn + " !\n\n")
}

//function for simplify rrsets update
func (FixDNS *Powerdns) PushArrayOfDNSRecordHostPtr(collection map[string][]string, ttl int) error{
    var rrSlice []interface{}
    for ip, domains := range collection {
        var recordSlice []interface{}
        ipAddressSlice := strings.Split(ip, ".")
        ptrRecord := ipAddressSlice[3] + "." + ipAddressSlice[2] + "." + ipAddressSlice[1] + "." + ipAddressSlice[0] + ".in-addr.arpa."
        ptrDomain := ipAddressSlice[2] + "." + ipAddressSlice[1] + "." + ipAddressSlice[0] + ".in-addr.arpa"
        for _, hostFqdn := range domains{
          record := Record{
            Content:  hostFqdn,
            Disabled: false,
            Name:     ptrRecord,
            TTL:      ttl,
            DType:    "PTR",
        }
        recordSlice = append(recordSlice, record)
    }

    rrSet := RrSet{
        Name:       ptrRecord,
        DType:      "PTR",
        TTL:        ttl,
        ChangeType: "REPLACE",
        Records:    recordSlice,
    }
    rrSlice = append(rrSlice, rrSet)

    update := make(map[string]interface{})
    update["rrsets"] = rrSlice
    jsonText, err := json.Marshal(update)
    topDomain, err := FixDNS.GetTopDomain(ptrDomain)
    if err != nil {
        fmt.Println("PDNS.UpdateRec: Could not get topdomain, reverting to domain: " + ptrDomain)
        fmt.Println(err)
        topDomain = ptrDomain
    }
    err = FixDNS.Patch("zones/"+topDomain, jsonText)
    if err != nil {
        fmt.Println("PDNS.UpdateRec: Error updating record at path zones/" + topDomain + " ,content:" + string(jsonText))
        fmt.Println(err)
        return err
    }
    fmt.Println("Updated PTR record, domain: " + ptrDomain + ", content: " + ptrRecord + " !\n\n")
}
return nil
}