package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"encoding/json"
	"io/ioutil"

	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type ExecutorStatus struct {
	Computer []struct {
		DisplayName string `json:"displayName"`
		Offline     bool   `json:"offline"`
	} `json:"computer"`
}

func getExecutorStatus(url string) (ExecutorStatus, error) {
	var executorStatus ExecutorStatus
	var err error

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/computer/api/json", url), nil)
	httpClient := http.Client{}
	res, err := httpClient.Do(req)

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &executorStatus)

	return executorStatus, err
}

func main() {
	var jenkinsHost string
	var pollTimer int
	flag.StringVar(&jenkinsHost, "urls", "", "remote Jenkins URLs - comma separated")
	flag.IntVar(&pollTimer, "pollDelay", 60, "specifies the delay in seconds between polling")
	flag.Parse()

	if len(jenkinsHost) == 0 {
		fmt.Println("The -urls flag is required - supply a comma separated list")
		return
	}

	agentStatus := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "online_status",
		Help: "gives a Jenkins build executor is online",
	}, []string{"node", "url"})

	prometheus.MustRegister(agentStatus)

	var hosts []string
	parts := strings.Split(jenkinsHost, ",")
	for _, part := range parts {
		hosts = append(hosts, strings.Trim(part, " "))
	}

	// Periodically sample build agents.
	go func() {

		for {
			for _, jenkinsHost := range hosts {
				status, err := getExecutorStatus(jenkinsHost)
				fmt.Println(status, err)
				for _, computer := range status.Computer {

					var online float64
					if computer.Offline == false {
						online = 1
					}

					labels := map[string]string{
						"node": computer.DisplayName,
						"url":  jenkinsHost,
					}

					agentStatus.With(labels).Set(online)
				}
			}
			time.Sleep(time.Duration(time.Duration(pollTimer) * time.Second))
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":9001", nil))
}
