package tempestapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"tempest_exporter/tempestudp"

	"github.com/prometheus/client_golang/prometheus"
)

type Client struct {
	token string
}

func NewClient(token string) Client {
	return Client{token: token}
}

type Station struct {
	Name         string
	StationID    int
	deviceID     int
	serialNumber string
	CreatedAt    time.Time
}

func (c Client) ListStations(ctx context.Context) ([]Station, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://swd.weatherflow.com/swd/rest/stations?token="+c.token, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Stations []struct {
			CreatedEpoch int64 `json:"created_epoch"`
			Devices      []struct {
				DeviceID     int    `json:"device_id"`
				DeviceType   string `json:"device_type"`
				SerialNumber string `json:"serial_number"`
			} `json:"devices"`
			Name      string `json:"name"`
			StationID int    `json:"station_id"`
		} `json:"stations"`
		Status struct {
			StatusCode    int    `json:"status_code"`
			StatusMessage string `json:"status_message"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var out []Station
	for _, station := range data.Stations {
		var deviceId int
		var instance string
		for _, dev := range station.Devices {
			if dev.DeviceType == "ST" {
				deviceId = dev.DeviceID
				instance = dev.SerialNumber
			}
		}

		if deviceId != 0 && instance != "" {
			out = append(out, Station{
				Name:         station.Name,
				deviceID:     deviceId,
				serialNumber: instance,
				StationID:    station.StationID,
				CreatedAt:    time.Unix(station.CreatedEpoch, 0),
			})
		}
	}
	return out, nil
}

func (c Client) GetObservations(ctx context.Context, station Station, startAt time.Time, endAt time.Time) ([]prometheus.Metric, error) {
	url := fmt.Sprintf("https://swd.weatherflow.com/swd/rest/observations/device/%d?token=%s&time_start=%d&time_end=%d", station.deviceID, c.token, startAt.Unix(), endAt.Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	report, err := tempestudp.ParseReport(bytes)
	if err != nil {
		log.Printf("read %s", string(bytes))
		return nil, err
	}

	switch r := report.(type) {
	case *tempestudp.TempestObservationReport:
		r.SerialNumber = station.serialNumber
	default:
		log.Fatalf("unhandled report type")
	}

	metrics := report.Metrics()
	return metrics, nil
}
