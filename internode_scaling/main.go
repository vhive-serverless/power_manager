package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"sync"
	"time"

	util "github.com/vhive-serverless/power_manager"
	powermanager "github.com/vhive-serverless/vhive/power_manager"
)

func main() {
	file, err := os.Create("metrics1.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.Write(append([]string{"startTime", "endTime", "spinningLatency", "sleepingLatency"}))
	if err != nil {
		fmt.Printf("Error writing metrics to the CSV file: %v\n", err)
	}

	ch := make(chan []string)
	ch_latency_spinning := make(chan int64)
	ch_latency_sleeping := make(chan int64)

	var wg sync.WaitGroup
	var wgConcurrent sync.WaitGroup

	wg.Add(1)
	go util.WriteToCSV(writer, ch, &wg)

	frequencies := map[string]int64{
		util.LowFrequencyPowerProfile:  1200,
		util.HighFrequencyPowerProfile: 2400,
	} // for 50/50, need to manually tune the frequency of the individual node
	
	for powerProfile, freq := range frequencies {
		err := powermanager.SetPowerProfileToNode(powerProfile, util.Node1Name, freq, freq)
		if err != nil {
			fmt.Printf(fmt.Sprintf("Error setting up power profile for node1: %+v", err))
		}
		err = powermanager.SetPowerProfileToNode(powerProfile, util.Node2Name, freq, freq)
		if err != nil {
			fmt.Printf(fmt.Sprintf("Error setting up power profile for node2: %+v", err))
		}

		now := time.Now()
		for time.Since(now) < (time.Minute * 1) {
			wgConcurrent.Add(10) 
			
			go util.InvokeConcurrently(5, util.SleepingURL, ch, ch_latency_spinning, ch_latency_sleeping, false, &wgConcurrent)
			go util.InvokeConcurrently(5, util.SpinningURL, ch, ch_latency_spinning, ch_latency_sleeping, true, &wgConcurrent)

			time.Sleep(1 * time.Second) // Wait for 1 second before invoking again
		}

		err = writer.Write(append([]string{"-", "-", "-", "-"}))
		if err != nil {
			fmt.Printf("Error writing metrics to the CSV file: %v\n", err)
		}
		time.Sleep(1 * time.Second) 
		fmt.Println("done")
	}
	wgConcurrent.Wait() // Wait for all invocation goroutines to complete

	close(ch)
	close(ch_latency_spinning)
	close(ch_latency_sleeping)

	wg.Wait()
}