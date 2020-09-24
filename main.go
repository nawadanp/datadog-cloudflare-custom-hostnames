package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	cloudflare "github.com/cloudflare/cloudflare-go"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type metric struct {
	Metric string   `json:"metric"`
	Type   string   `json:"type"`
	Points []points `json:"points"`
	Tags   []string `json:"tags"`
}

type series struct {
	Series []metric `json:"series"`
}

type points [2]float64

const (
	dataDogAPIURL = "https://api.datadoghq.com/api/v1"
)

func main() {
	cfAPIKey := os.Getenv("CF_API_KEY")
	if cfAPIKey == "" {
		log.Fatalf("%s should be defined", "CF_API_KEY")
	}
	cfLogin := os.Getenv("CF_LOGIN")
	if cfAPIKey == "" {
		log.Fatalf("%s should be defined", "CF_LOGIN")
	}
	cfZone := os.Getenv("CF_ZONE")
	if cfAPIKey == "" {
		log.Fatalf("%s should be defined", "CF_ZONE")
	}
	cfCHQuota, err := strconv.Atoi(os.Getenv("CF_CH_QUOTA"))
	if err != nil {
		log.Fatal(err)
	}
	ddAPIKey := os.Getenv("DD_API_KEY")
	if cfAPIKey == "" {
		log.Fatalf("%s should be defined", "DD_API_KEY")
	}

	// Construct a new API object
	api, err := cloudflare.New(cfAPIKey, cfLogin)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch the zone ID
	id, err := api.ZoneIDByName(cfZone)
	if err != nil {
		log.Fatal(err)
	}

	var hostname cloudflare.CustomHostname
	_, status, _ := api.CustomHostnames(id, 1, hostname)

	var customHostnames points
	customHostnames[0] = float64(time.Now().Unix())
	customHostnames[1] = float64(status.Total)

	var customHostnamesQuota points
	customHostnamesQuota[0] = float64(time.Now().Unix())
	customHostnamesQuota[1] = float64(cfCHQuota)

	var tag []string
	tag = append(tag, "cf_domain:"+cfZone)

	CFCustomHostnames := metric{"custom.cloudflare.custom_hostname", "gauge", []points{customHostnames}, tag}
	CFCustomHostnamesQuota := metric{"custom.cloudflare.custom_hostname_quota", "gauge", []points{customHostnamesQuota}, tag}

	var series series
	series.Series = append(series.Series, CFCustomHostnames)
	series.Series = append(series.Series, CFCustomHostnamesQuota)

	err = pushToDatadog(ddAPIKey, series)
	if err != nil {
		fmt.Println(err)
	}
}

func pushToDatadog(key string, series series) error {
	// Build the query
	url := dataDogAPIURL + "/series?api_key=" + key
	jsonStr, _ := json.Marshal(series)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	// Run the query
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		e := fmt.Errorf("HTTP Request Error : %d", err)
		return e
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 {
		e := fmt.Errorf("Datadog API Error : %d", resp.StatusCode)
		return e
	}
	return nil
}

