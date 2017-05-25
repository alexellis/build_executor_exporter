package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// ExecutorStatus from Jenkins public API
type ExecutorStatus struct {
	Computer []struct {
		DisplayName        string `json:"displayName"`
		Offline            bool   `json:"offline"`
		TemporarilyOffline bool   `json:"temporarilyOffline"`
	} `json:"computer"`
}

type remoteCollection struct {
	node               string
	url                string
	online             float64
	temporarilyOffline float64
}

type executorCollector struct {
	onlineStatus             *prometheus.Desc
	temporarilyOfflineStatus *prometheus.Desc
	URLs                     []string
}

func main() {
	var jenkinsHost string
	var oneShot bool

	flag.StringVar(&jenkinsHost, "urls", "", "remote Jenkins URLs - comma separated")
	flag.BoolVar(&oneShot, "oneShot", false, "print to stdout and exit")

	flag.Parse()

	if len(jenkinsHost) == 0 {
		fmt.Println("The -urls flag is required - supply a comma separated list")
		return
	}

	URLs := getHosts(jenkinsHost)
	collector := newExecutorCollector(URLs)
	prometheus.Register(collector)

	if oneShot {
		req := httptest.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()

		var handler http.Handler
		handler = promhttp.Handler()
		handler.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(body))
	} else {
		http.Handle("/metrics", promhttp.Handler())
		log.Fatal(http.ListenAndServe(":9001", nil))
	}
}

func getHosts(jenkinsHost string) []string {
	var hosts []string
	parts := strings.Split(jenkinsHost, ",")
	for _, part := range parts {
		hosts = append(hosts, strings.Trim(part, " "))
	}
	return hosts
}

// NewExecutorCollector creates new executorCollector
func newExecutorCollector(URLs []string) *executorCollector {
	c := executorCollector{
		URLs: URLs,
	}
	c.onlineStatus = prometheus.NewDesc("online_status", "whether a node is online", []string{"name", "url"}, prometheus.Labels{})
	c.temporarilyOfflineStatus = prometheus.NewDesc("temporarily_offline_status", "whether a node is temporarily offline", []string{"name", "url"}, prometheus.Labels{})

	return &c
}

// Describe sends the super-set of all possible descriptors of metrics
// collected by this Collector to the provided channel and returns once
// the last descriptor has been sent.
func (c *executorCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.onlineStatus
}

// Collect is called by the Prometheus registry when collecting
// metrics.
func (c *executorCollector) Collect(ch chan<- prometheus.Metric) {
	for _, url := range c.URLs {
		results, err := remoteCollect(url)
		if err != nil {
			fmt.Println(err)
		} else {
			for _, result := range results {
				ch <- prometheus.MustNewConstMetric(c.onlineStatus, prometheus.GaugeValue, result.online, result.node, url)
				ch <- prometheus.MustNewConstMetric(c.temporarilyOfflineStatus, prometheus.GaugeValue, result.temporarilyOffline, result.node, url)
			}
		}
	}
}

func remoteCollect(url string) ([]remoteCollection, error) {
	var results []remoteCollection
	var err error

	status, err := getExecutorStatus(url)

	if err == nil {
		for _, computer := range status.Computer {
			result := remoteCollection{
				node:               computer.DisplayName,
				online:             0,
				temporarilyOffline: 0,
			}

			if computer.Offline == false {
				result.online = 1
			}

			if computer.TemporarilyOffline {
				result.temporarilyOffline = 1
			}

			results = append(results, result)
		}
	}

	return results, err
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
