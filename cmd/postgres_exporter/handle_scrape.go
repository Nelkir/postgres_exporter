package main

import (
  "net/http"
	"bytes"
  "fmt"
	"time"
  
	"github.com/evanj/concurrentlimit"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/expfmt"
	"github.com/go-kit/log/level"
)

type MetricScrapeHandler struct {
  metrics []byte
}

func (metric_handler *MetricScrapeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { 
  w.Write(metric_handler.metrics)
}

func GetHandler(withInterval bool) http.Handler {
  if withInterval {
    handler := MetricScrapeHandler{}
    interval, err := time.ParseDuration(*gatherInterval)
    if err != nil {
      level.Error(logger).Log("msg", "Can't parse gather interval", "err", err)      
    } else {
      ticker := time.NewTicker(interval)
   
      go handler.intervalMetricUpdater(ticker) 
      
      return &handler 
    }
  } 

  limiter := concurrentlimit.New(*concurrentRequestLimit)
  return concurrentlimit.Handler(limiter, promhttp.Handler())
}

func updateMetrics() []byte {
  metric_families, err := prometheus.DefaultGatherer.Gather()
  if err != nil {
    level.Error(logger).Log("msg", "Can't gather metrics", "err", err)
  }

  buf:= new(bytes.Buffer)
  for _, mf := range metric_families {
    expfmt.MetricFamilyToText(buf, mf)
  }
  expfmt.FinalizeOpenMetrics(buf)

  return buf.Bytes()
}

func (metric_handler *MetricScrapeHandler) intervalMetricUpdater(ticker *time.Ticker) {
  metric_handler.metrics = updateMetrics()
  for {
    select {
      case <-ticker.C:
        level.Info(logger).Log("msg", fmt.Sprintf("Updatig metrics %s", time.Now().Format(time.RFC3339)))  
        metric_handler.metrics = updateMetrics() 
    }
  }
}
