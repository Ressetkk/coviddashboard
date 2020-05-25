package main

import (
	"bytes"
	"crypto/md5"
	"encoding/csv"
	"fmt"
	client "github.com/influxdata/influxdb1-client/v2"
	"github.com/mmcloughlin/geohash"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	dataURL = "https://coronadatascraper.com/timeseries.csv"
	db = "covid"
	measurement = "cases"
)
var (
	headerTypes = map[string]string{
		"name": "tag",
		"country": "tag",
		"level": "tag",
		"county": "tag",
		"population": "field",
		"lat": "field",
		"long": "field",
		"cases": "field",
		"deaths": "field",
		"recovered": "field",
		"active": "field",
		"date": "timestamp"}
	fieldTypes = map[string]string{
		"population": "int",
		"lat": "float",
		"long": "float",
		"cases": "int",
		"deaths": "int",
		"recovered": "int",
		"active": "int",
	}
	lastSum []byte
)
func main() {
	server := os.Getenv("INFLUXDB_SERVER")
	if server == "" {
		server = "http://localhost:8086"
	}
	cl, err := client.NewHTTPClient(client.HTTPConfig{Addr: server})
	if err != nil {
		log.Fatalf("Could not create connection to InfluxDB: %v", err)
	}

	var dbExists bool
	dbsResp, err := cl.Query(client.Query{Command: "SHOW DATABASES"})
	if err != nil {
		log.Fatalf("Invalid server address: %s", err)
	}
	for _, v := range dbsResp.Results[0].Series[0].Values {
		dbName := v[0].(string)
		if db == dbName {
			dbExists = true
			break
		}
	}
	if dbExists {
		_, _ = cl.Query(client.Query{Command: fmt.Sprintf("DROP DATABASE %s", db)})
		_, _ = cl.Query(client.Query{Command: fmt.Sprintf("CREATE DATABASE %s", db)})
	} else {
		_, _ = cl.Query(client.Query{Command: fmt.Sprintf("CREATE DATABASE %s", db)})
	}

	checkForUpdates := func() {
		r, err := getData()
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			log.Printf("Error getting data: %v. Retrying in the next tick...", err)
			return
		}
		newSum := calculateMD5(buf)
		if !bytes.Equal(lastSum, newSum) {
			go updateData(cl, bytes.NewReader(buf))
			lastSum = newSum
		}
	}
	checkForUpdates()
	ticker := time.NewTicker(5*time.Minute)
	for {
		select {
		case <- ticker.C:
			checkForUpdates()
		}
	}
}

func updateData(cl client.Client, r io.Reader) {
	log.Printf("Updating data at %s.", time.Now())

	f := csv.NewReader(r)
	headers, _ := f.Read()
	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{Database: db})
	bpSize := 0
	totalSize := 0
	var timestamp time.Time
	var lat, long float64
	for {
		tags := make(map[string]string)
		fields := make(map[string]interface{})

		row, err := f.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("CSV read error: %v\n", err)
		}
		for col, val := range row {
			if headerTypes[headers[col]] == "tag" {
				tags[headers[col]] = val
			} else if headerTypes[headers[col]] == "field" {
				if fieldTypes[headers[col]] == "int" {
					var v int
					if val != "" {
						v,  _ = strconv.Atoi(val)
					}
					fields[headers[col]] = v
				} else if fieldTypes[headers[col]] == "float" {
					var v float64
					if val != "" {
						v, _ = strconv.ParseFloat(val, 64)
					}
					if headers[col] == "lat" {
						lat = v
					}
					if headers[col] == "long" {
						long = v
					}
					fields[headers[col]] = v
				}
			} else if headerTypes[headers[col]] == "timestamp" {
				t, _ := time.Parse("2006-01-02", val)
				timestamp = t
			}
		}
		tags["geohash"] = geohash.Encode(lat, long)
		point, err := client.NewPoint(measurement, tags, fields, timestamp)
		if err != nil {
			log.Fatalf("Could not create point: %v\n", err)
		}
		bp.AddPoint(point)
		bpSize++
		totalSize++
		if bpSize == 1000 {
			cl.Query(client.Query{Command: fmt.Sprintf("DROP MEASUREMENT %s", measurement)})
			if err := cl.Write(bp); err != nil {
				log.Fatalf("Could not write batch to InfluxDB: %v\n", err)
			}
			bp, _ = client.NewBatchPoints(client.BatchPointsConfig{Database: db})
			bpSize = 0
		}
	}
	log.Println("Done.")
}

func getData() (io.Reader, error) {
	resp, err := http.Get(dataURL)
	if err != nil {
		return nil, fmt.Errorf("download error: %w", err)
	}
	return resp.Body, nil
}

func calculateMD5(b []byte) []byte {
	hash := md5.New()
	_, err := io.Copy(hash, bytes.NewReader(b))
	if err != nil {
		log.Printf("MD5 calculation error: %v\n", err)
	}
	return hash.Sum(nil)
}
