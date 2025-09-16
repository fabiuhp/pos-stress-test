package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	cfg, err := parseFlags()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		flag.Usage()
		os.Exit(1)
	}

	report := runLoadTest(cfg)
	printReport(report)
}

type config struct {
	url         string
	requests    int
	concurrency int
	timeout     time.Duration
}

type report struct {
	duration       time.Duration
	totalRequests  int32
	success200     int32
	statusCounters map[int]int32
	errors         int32
}

func parseFlags() (*config, error) {
	url := flag.String("url", "", "URL do serviço a ser testado")
	requests := flag.Int("requests", 0, "Número total de requests a serem realizados")
	concurrency := flag.Int("concurrency", 1, "Número de chamadas simultâneas")
	timeout := flag.Duration("timeout", 30*time.Second, "Tempo máximo para aguardar cada request (ex: 10s, 1m)")

	flag.Parse()

	if *url == "" {
		return nil, errors.New("o parâmetro --url é obrigatório")
	}

	if *requests <= 0 {
		return nil, errors.New("o parâmetro --requests deve ser maior que zero")
	}

	if *concurrency <= 0 {
		return nil, errors.New("o parâmetro --concurrency deve ser maior que zero")
	}

	if *concurrency > *requests {
		*concurrency = *requests
	}

	return &config{
		url:         *url,
		requests:    *requests,
		concurrency: *concurrency,
		timeout:     *timeout,
	}, nil
}

func runLoadTest(cfg *config) report {
	client := &http.Client{Timeout: cfg.timeout}

	var success200 int32
	var totalRequests int32
	var errorsCount int32

	statusCounters := make(map[int]int32)
	var statusMu sync.Mutex

	var wg sync.WaitGroup
	jobs := make(chan struct{}, cfg.requests)

	worker := func() {
		defer wg.Done()
		for range jobs {
			atomic.AddInt32(&totalRequests, 1)
			status, err := performRequest(client, cfg.url)
			if err != nil {
				atomic.AddInt32(&errorsCount, 1)
				continue
			}

			if status == http.StatusOK {
				atomic.AddInt32(&success200, 1)
			}

			statusMu.Lock()
			statusCounters[status]++
			statusMu.Unlock()
		}
	}

	start := time.Now()

	wg.Add(cfg.concurrency)
	for i := 0; i < cfg.concurrency; i++ {
		go worker()
	}

	for i := 0; i < cfg.requests; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	wg.Wait()

	duration := time.Since(start)

	return report{
		duration:       duration,
		totalRequests:  totalRequests,
		success200:     success200,
		statusCounters: statusCounters,
		errors:         errorsCount,
	}
}

func performRequest(client *http.Client, url string) (int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func printReport(r report) {
	fmt.Println("===== Relatório de Teste de Carga =====")
	fmt.Printf("Tempo total: %s\n", r.duration)
	fmt.Printf("Total de requests realizados: %d\n", r.totalRequests)
	fmt.Printf("Requests com status 200: %d\n", r.success200)

	fmt.Println("Distribuição de status HTTP:")
	if len(r.statusCounters) == 0 {
		fmt.Println("  Nenhum status HTTP foi registrado (verifique erros de conexão).")
	} else {
		for status, count := range r.statusCounters {
			fmt.Printf("  %d: %d\n", status, count)
		}
	}

	if r.errors > 0 {
		fmt.Printf("Erros de requisição: %d\n", r.errors)
	}
}
