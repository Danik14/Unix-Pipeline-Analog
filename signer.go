package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func ExecutePipeline(jobs ...job) {
	//creating pointer for my wait group
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	//starting to loop through jobs
	for _, currentJob := range jobs {
		wg.Add(1)

		out := make(chan interface{})

		//creating goroutine for each job
		//in order to save time
		go func(someJob job, in, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)

			someJob(in, out)
		}(currentJob, in, out, wg)

		//"making pipe"
		in = out
	}

	//blocking until function ends
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	chanStruct := make(chan struct{}, 1)

	for currentData := range in {
		currentData := fmt.Sprintf("%v", currentData)
		uniChan := make(chan string, 1)

		wg.Add(1)
		go func(data string, out chan<- string, chanStruct chan struct{}) {
			defer wg.Done()

			chanStruct <- struct{}{}
			dataHash := DataSignerMd5(data)
			<-chanStruct
			out <- DataSignerCrc32(dataHash)
		}(currentData, uniChan, chanStruct)

		wg.Add(1)
		go func(data string, in <-chan string, out chan<- interface{}) {
			defer wg.Done()

			out <- DataSignerCrc32(data) + "~" + <-in
		}(currentData, uniChan, out)
	}

	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	wg2 := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	for currentData := range in {
		result := make([]string, 6)

		for i := 0; i < 6; i++ {
			wg2.Add(1)
			go func(i int, data string) {
				defer wg2.Done()

				dataHash := DataSignerCrc32(strconv.Itoa(i) + data)
				mu.Lock()

				result[i] = dataHash
				mu.Unlock()
			}(i, currentData.(string))
		}

		wg.Add(1)
		go func(out chan<- interface{}) {
			defer wg.Done()

			wg2.Wait()
			out <- strings.Join(result, "")
		}(out)
	}

	wg.Wait()
}

func CombineResults(in, out chan interface{}) {
	var result []string

	for currentData := range in {
		result = append(result, currentData.(string))
	}

	sort.Strings(result)
	out <- strings.Join(result, "_")
}
