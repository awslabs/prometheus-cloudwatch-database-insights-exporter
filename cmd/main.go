package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/collector"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/manager/region"
	"github.com/awslabs/prometheus-cloudwatch-database-insights-exporter/pkg/utils"
)

const (
	// MaxInstanceIdentifiers defines the maximum number of instance identifiers
	// allowed in the ?identifiers query parameter to prevent service overload
	MaxInstanceIdentifiers = 5
)

func main() {
	log.Println("[MAIN] Starting Database Insights Exporter")

	cfg, err := utils.LoadConfig("config.yml")
	if err != nil {
		log.Fatalf("[MAIN] Error loading configuration: %v", err)
	}

	factory := region.NewRegionManagerFactory()
	regionManager, err := factory.CreateRegionManager(cfg)
	if err != nil {
		log.Fatalf("[MAIN] Error creating region manager: %v", err)
	}

	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsHandler(w, r, regionManager)
	})

	log.Printf("[MAIN] Starting HTTP server on port %d", cfg.Export.Port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.Export.Port), nil))
}

func metricsHandler(w http.ResponseWriter, r *http.Request, regionManager region.RegionManager) {
	start := time.Now()

	query := r.URL.Query()
	instanceIdentifiers := query.Get("identifiers")

	var collectorInstance prometheus.Collector
	if instanceIdentifiers != "" {
		identifiers := strings.Split(instanceIdentifiers, ",")
		for i, id := range identifiers {
			identifiers[i] = strings.TrimSpace(id)
		}

		if len(identifiers) > MaxInstanceIdentifiers {
			log.Printf("[HTTP] %s %s - Too many identifiers: %d (max: %d)", r.Method, r.URL.Path, len(identifiers), MaxInstanceIdentifiers)
			http.Error(w, fmt.Sprintf("Too many instance identifiers provided. Maximum allowed: %d, provided: %d", MaxInstanceIdentifiers, len(identifiers)), http.StatusBadRequest)
			return
		}

		log.Printf("[HTTP] %s %s - Filtering for instance: %s", r.Method, r.URL.Path, instanceIdentifiers)
		collectorInstance = collector.NewFilteredCollector(regionManager, identifiers)
	} else {
		log.Printf("[HTTP] %s %s - All instances", r.Method, r.URL.Path)
		collectorInstance = collector.NewCollector(regionManager)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collectorInstance)

	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	handler.ServeHTTP(w, r)

	duration := time.Since(start)
	log.Printf("[HTTP] %s %s - Completed in %v", r.Method, r.URL.Path, duration)
}
