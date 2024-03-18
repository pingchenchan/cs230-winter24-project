package main

import (
    "context"
    "fmt"
    "github.com/influxdata/influxdb-client-go/v2"
    "log"
	"os"

)

func QueryData(zone string, influxClient influxdb2.Client) (float64, error){
    // Get the organization from environment variables
    influxdbOrg := os.Getenv("INFLUXDB_ORG")
	// influxdbOrg := "cs230"
	// fmt.Printf("run QueryData %s\n", zone)

    // Create a new query to get data from the bucket
    query := fmt.Sprintf(`from(bucket: "server")
      |> range(start: -10s, stop: now())
      |> filter(fn: (r) => r["_measurement"] == "kafka_consumer")
      |> filter(fn: (r) => r["server"] == "%s")
      |> filter(fn: (r) => r["_field"] == "cpu_usage")
      |> aggregateWindow(every: 10s, fn: mean, createEmpty: false)
      |> yield(name: "mean")`, zone)

   // Get the query API from the client
   queryAPI := influxClient.QueryAPI(influxdbOrg)
   var meanCPUUsage float64 = DEFAULT_CPU_USAGE
   // Execute the query
   result, err := queryAPI.Query(context.Background(), query)
   if err != nil {
	   log.Printf("Error querying data: %s", err)
	   return meanCPUUsage, nil
   }
  
	

	for result.Next() {
		record := result.Record()
		// fmt.Printf("Record: %v\n", record.Values())
		if value, ok := record.ValueByKey("_value").(float64); ok {
			meanCPUUsage = value
		}
		//   fmt.Printf(zone, " CPU Usage: ", meanCPUUsage)
		//   fmt.Printf("\n")
	}

	if result.Err() != nil {
		log.Printf("Query error: %s", result.Err())
		return meanCPUUsage, result.Err()
	}
  
	  return meanCPUUsage, nil
  }