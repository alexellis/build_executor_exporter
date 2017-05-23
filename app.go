package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ExecutorStatus from Jenkins public API
type ExecutorStatus struct {
	Computer []struct {
		DisplayName string `json:"displayName"`
		Offline     bool   `json:"offline"`
	} `json:"computer"`
}

func main() {
	var jenkinsHost string
	var pollTimer int

	var oneShot bool

	flag.StringVar(&jenkinsHost, "urls", "", "remote Jenkins URLs - comma separated")
	flag.IntVar(&pollTimer, "pollDelay", 60, "specifies the delay in seconds between polling")
	flag.BoolVar(&oneShot, "oneShot", false, "print to stdout and exit")

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
	hosts := getHosts(jenkinsHost)

	if oneShot {
		collect(hosts, agentStatus)

		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		var handler http.Handler
		handler = promhttp.Handler()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		fmt.Println(string(body))
	} else {
		// Periodically sample build agents.
		go func() {
			for {
				collect(hosts, agentStatus)

				time.Sleep(time.Duration(time.Duration(pollTimer) * time.Second))
			}
		}()

		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":9001", nil))
	}
}

func collect(hosts []string, agentStatus *prometheus.GaugeVec) {
	for _, jenkinsHost := range hosts {
		status, err := getExecutorStatus(jenkinsHost)
		if err != nil {
			fmt.Println(err)
		}
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
}

func getExecutorStatus(url string) (ExecutorStatus, error) {
	var executorStatus ExecutorStatus
	var err error

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/computer/api/json", url), nil)
	httpClient := http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		return executorStatus, err
	}

	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	err = json.Unmarshal(resBody, &executorStatus)

	return executorStatus, err
}

func getHosts(jenkinsHost string) []string {
	var hosts []string
	parts := strings.Split(jenkinsHost, ",")
	for _, part := range parts {
		hosts = append(hosts, strings.Trim(part, " "))
	}
	return hosts
}
