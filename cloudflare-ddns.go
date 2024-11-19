package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudflare/cloudflare-go"
)

// Config holds the configuration for the DDNS client
type Config struct {
	APIKey     string
	APIMail    string
	ZoneID     string
	RecordID   string
	RecordName string
	Interval   time.Duration
}

// GetPublicIP fetches the public IP address of the client
func GetPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org") //This URL returns your ip as a string
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var ip string
	_, err = fmt.Fscan(resp.Body, &ip)
	if err != nil {
		return "", err
	}

	fmt.Println("Get Ip res ", ip)
	return ip, nil
}

// GetDNSRecordID fetches the DNS record ID for the given record name
func GetDNSRecordID(api *cloudflare.API, config Config) (string, error) {
	ctx := context.Background()
	resourceContainer := cloudflare.ZoneIdentifier(config.ZoneID)
	records, _, err := api.ListDNSRecords(ctx, resourceContainer, cloudflare.ListDNSRecordsParams{Name: config.RecordName})
	if err != nil {
		return "", err
	}

	for _, record := range records {
		fmt.Println("The recordID is ", record.ID)
		if record.Name == config.RecordName {
			return record.ID, nil
		}
	}

	return "", fmt.Errorf("record not found")
}

// UpdateDNSRecord updates the DNS record with the given IP
func UpdateDNSRecord(api *cloudflare.API, config Config, ip string) error {
	ctx := context.Background()

	params := cloudflare.UpdateDNSRecordParams{
		Type:    "A",
		Name:    config.RecordName,
		Content: ip,
		ID:      config.RecordID,
	}

	resourceContainer := cloudflare.ZoneIdentifier(config.ZoneID)

	_, err := api.UpdateDNSRecord(ctx, resourceContainer, params)
	if err != nil {
		return err
	}

	return nil

}

func main() {
	config := Config{
		APIKey:     os.Getenv("CLOUDFLARE_API_KEY"),
		APIMail:    os.Getenv("CLOUDFLARE_API_MAIL"),
		ZoneID:     os.Getenv("CLOUDFLARE_ZONE_ID"),
		RecordID:   os.Getenv("CLOUDFLARE_RECORD_ID"),
		RecordName: os.Getenv("CLOUDFLARE_RECORD_NAME"),
		Interval:   5 * time.Minute, // Adjust as needed
	}

	//fmt.Printf("%s: %s\n", "apikey", config.APIKey)
	//fmt.Printf("%s: %s\n", "apimail", config.APIMail)

	//api, err := cloudflare.NewWithAPIToken(config.APIKey)
	api, err := cloudflare.New(config.APIKey, config.APIMail)
	if err != nil {
		log.Fatalf("Error creating Cloudflare client: %v", err)
	}

	for {
		ip, err := GetPublicIP()
		if err != nil {

			log.Printf("Error fetching public IP: %v", err)
			time.Sleep(config.Interval)
			continue
		}
		recordId, err := GetDNSRecordID(api, config)
		if err != nil {
			log.Fatalf("Error getting DNS Record Id: %v", err)
		}

		//fmt.Println("Found Record Id ", recordId)
		config.RecordID = recordId

		err = UpdateDNSRecord(api, config, ip)
		if err != nil {
			log.Printf("error in UpdateDNSRecord: %v", err)
		} else {
			log.Printf("Successfully updated DNS record to %s", ip)
		}

		config.RecordName = "*." + config.RecordName
		recordId, err = GetDNSRecordID(api, config)
		if err != nil {
			log.Fatalf("Error getting DNS Record Id: %v", err)
		}

		//fmt.Println("Found Record Id ", recordId)
		config.RecordID = recordId

		err = UpdateDNSRecord(api, config, ip)
		if err != nil {
			log.Printf("error in UpdateDNSRecord: %v", err)
		} else {
			log.Printf("Successfully updated DNS record to %s", ip)
		}

		time.Sleep(config.Interval)
	}
}
