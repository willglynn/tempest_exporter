package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"tempest_exporter/tempest"
	"tempest_exporter/tempestapi"
	"tempest_exporter/tempestudp"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/prometheus/common/expfmt"
)

type collector struct {
	outbox <-chan prometheus.Metric
}

func (c collector) Describe(descs chan<- *prometheus.Desc) {
	for _, desc := range tempest.All {
		descs <- desc
	}
}

func (c collector) Collect(metrics chan<- prometheus.Metric) {
	for {
		select {
		case m := <-c.outbox:
			metrics <- m
		default:
			return
		}
	}
}

func main() {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt)
	defer done()

	token := os.Getenv("TOKEN")
	if token != "" {
		export(ctx, token)
	} else {
		listenAndPush(ctx)
	}
}

func listenAndPush(ctx context.Context) {
	pushUrl := os.Getenv("PUSH_URL")
	if pushUrl == "" {
		log.Fatal("PUSH_URL must be specified")
	}
	jobName := os.Getenv("JOB_NAME")
	if jobName == "" {
		jobName = "tempest"
	}
	log.Printf("pushing to %q with job name %q", pushUrl, jobName)

	more := make(chan bool, 1)
	outbox := make(chan prometheus.Metric, 1000)
	go func() {
		c := &collector{outbox}
		pusher := push.New(pushUrl, jobName).Collector(c).Format(expfmt.FmtText)

		for {
			select {
			case <-ctx.Done():
				return
			case <-more:
				if err := pusher.Add(); err != nil {
					log.Printf("error pushing: %v", err)
				}
			}
		}
	}()

	if err := listen(ctx, func(b []byte, addr *net.UDPAddr) error {
		log.Printf("UDP in: %s", string(b))
		report, err := tempestudp.ParseReport(b)
		if err != nil {
			log.Printf("error parsing report from %s: %s", addr, err)
		} else {
			for _, m := range report.Metrics() {
				outbox <- m
			}

			select {
			case more <- true:
				// success
			default:
				// already busy sending
			}
		}

		return nil
	}); err != nil {
		log.Fatal(err)
	}
}

func listen(ctx context.Context, rx func([]byte, *net.UDPAddr) error) error {
	sock, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   nil,
		Port: 50222,
	})
	if err != nil {
		return err
	}
	defer sock.Close()
	log.Printf("listening on UDP :50222")

	readErr := make(chan error, 1)

	// Start reading in the background
	go func() {
		buffer := make([]byte, 1500)
		for {
			n, addr, err := sock.ReadFromUDP(buffer)
			if err != nil {
				readErr <- err
				break
			}
			err = rx(buffer[:n], addr)
			if err != nil {
				readErr <- err
				break
			}
		}
		close(readErr)
	}()

	// Wait for reading to finish, or for our context to finish
	select {
	case err := <-readErr:
		return err

	case <-ctx.Done():
		return nil
	}
}

func export(ctx context.Context, token string) {
	client := tempestapi.NewClient(token)
	stations, err := client.ListStations(ctx)
	if err != nil {
		log.Fatalf("error listing stations: %v", err)
	}

	if len(stations) == 0 {
		log.Fatalf("no stations found")
	}

	log.Printf("found stations:")
	var startAt time.Time
	for _, station := range stations {
		log.Printf("  - %s (station #%d)", station.Name, station.StationID)
		if startAt.IsZero() || startAt.Before(station.CreatedAt) {
			startAt = station.CreatedAt
		}
	}

	n := 1

	var next time.Time
	cur := startAt
	for {
		var c dumpCollector

		for ; cur.Before(time.Now()) && len(c.metrics) < 200_000; cur = next {
			next = cur.AddDate(0, 0, 1) // for 1-minute observation frequency

			for _, station := range stations {
				log.Printf("fetching %s starting %s", station.Name, cur.Format(time.RFC3339))
				metrics, err := client.GetObservations(ctx, station, cur, next)
				if err != nil {
					log.Fatalf("error fetching %#v for %d-%d: %v", station, cur.Unix(), next.Unix(), err)
				}
				c.metrics = append(c.metrics, metrics...)
			}
		}

		if len(c.metrics) == 0 {
			break
		}

		r := prometheus.NewRegistry()
		r.MustRegister(&c)
		families, err := r.Gather()
		if err != nil {
			log.Fatalf("error gathering metrics: %v", err)
		}

		filename := fmt.Sprintf("tempest_%03d.txt.gz", n)
		log.Printf("writing %s", filename)
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("error opening output file: %v", err)
		}
		gzw := gzip.NewWriter(f)
		enc := expfmt.NewEncoder(gzw, expfmt.FmtText)
		for _, family := range families {
			if err := enc.Encode(family); err != nil {
				log.Fatalf("error encoding metrics: %v", err)
			}
		}
		if c, ok := enc.(io.Closer); ok {
			err = c.Close()
			if err != nil {
				log.Fatalf("error closing metric encoder: %v", err)
			}
		}
		if err := gzw.Close(); err != nil {
			log.Fatalf("error closing gzip writer: %v", err)
		}
		if err := f.Close(); err != nil {
			log.Fatalf("error closing output file: %v", err)
		}

		n = n + 1
	}
}

type dumpCollector struct {
	metrics []prometheus.Metric
}

func (d dumpCollector) Describe(descs chan<- *prometheus.Desc) {
}

func (d dumpCollector) Collect(metrics chan<- prometheus.Metric) {
	for _, metric := range d.metrics {
		metrics <- metric
	}
}
