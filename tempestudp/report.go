package tempestudp

import (
	"encoding/json"
	"fmt"
	"time"

	"tempest_exporter/tempest"

	"github.com/prometheus/client_golang/prometheus"
)

// Docs: https://weatherflow.github.io/Tempest/api/udp/v143/

type Report interface {
	Metrics() []prometheus.Metric
}

func ParseReport(bytes []byte) (Report, error) {
	var typ struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(bytes, &typ); err != nil {
		return nil, err
	}

	var data Report
	switch typ.Type {
	case "evt_precip":
		data = &rainStartReport{}
	case "evt_strike":
		data = &lightningStrikeReport{}
	case "rapid_wind":
		data = &rapidWindReport{}
	case "obs_st":
		data = &TempestObservationReport{}
	case "device_status":
		data = &deviceStatusReport{}
	case "hub_status":
		data = &hubStatusReport{}
	default:
		return nil, fmt.Errorf("unhandled message type: %q", typ.Type)
	}

	if err := json.Unmarshal(bytes, data); err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

type rainStartReport struct {
	SerialNumber string `json:"serial_number"`

	// "evt_precip"
	Type string `json:"type"`

	HubSn string `json:"hub_sn"`

	// 0	Time Epoch	Seconds
	Evt []float64 `json:"evt"`
}

func (r rainStartReport) Metrics() []prometheus.Metric {
	return nil
}

type lightningStrikeReport struct {
	SerialNumber string `json:"serial_number"`

	// "evt_strike"
	Type string `json:"type"`

	HubSn string `json:"hub_sn"`

	// 0	Time Epoch	Seconds
	// 1	Distance	km
	// 2	Energy
	Evt []float64 `json:"evt"`
}

func (r lightningStrikeReport) Metrics() []prometheus.Metric {
	return nil
}

type rapidWindReport struct {
	SerialNumber string `json:"serial_number"`

	// "rapid_wind"
	Type string `json:"type"`

	HubSn string    `json:"hub_sn"`
	Ob    []float64 `json:"ob"`
}

func (r rapidWindReport) Metrics() []prometheus.Metric {
	if len(r.Ob) != 3 {
		return nil
	}

	ts := int64(r.Ob[0])
	return withTime(ts, []prometheus.Metric{
		prometheus.MustNewConstMetric(tempest.Wind, prometheus.GaugeValue, r.Ob[1], r.SerialNumber, "rapid"),
		prometheus.MustNewConstMetric(tempest.WindDirection, prometheus.GaugeValue, r.Ob[2], r.SerialNumber),
	})
}

type TempestObservationReport struct {
	SerialNumber string `json:"serial_number"`

	// "obs_st"
	Type string `json:"type"`

	HubSn string `json:"hub_sn"`

	// 0	Time Epoch	Seconds
	// 1	Wind Lull (minimum 3 second sample)	m/s
	// 2	Wind Avg (average over report interval)	m/s
	// 3	Wind Gust (maximum 3 second sample)	m/s
	// 4	Wind Direction	Degrees
	// 5	Wind Sample Interval	seconds
	// 6	Station Pressure	MB
	// 7	Air Temperature	C
	// 8	Relative Humidity	%
	// 9	Illuminance	Lux
	// 10	UV	Index
	// 11	Solar Radiation	W/m^2
	// 12	Rain amount over previous minute	mm
	// 13	Precipitation Type	0 = none, 1 = rain, 2 = hail, 3 = rain + hail (experimental)
	// 14	Lightning Strike Avg Distance	km
	// 15	Lightning Strike Count
	// 16	Battery	Volts
	// 17	Report Interval	Minutes
	Obs [][]float64 `json:"obs"`

	FirmwareRevision int `json:"firmware_revision"`
}

func (r TempestObservationReport) Metrics() []prometheus.Metric {
	var out []prometheus.Metric

	for _, ob := range r.Obs {
		if len(ob) < 13 {
			continue
		}

		metrics := []prometheus.Metric{
			prometheus.MustNewConstMetric(tempest.Wind, prometheus.GaugeValue, ob[1], r.SerialNumber, "lull"),
			prometheus.MustNewConstMetric(tempest.Wind, prometheus.GaugeValue, ob[2], r.SerialNumber, "avg"),
			prometheus.MustNewConstMetric(tempest.Wind, prometheus.GaugeValue, ob[3], r.SerialNumber, "gust"),
			prometheus.MustNewConstMetric(tempest.WindDirection, prometheus.GaugeValue, ob[4], r.SerialNumber),
			prometheus.MustNewConstMetric(tempest.Pressure, prometheus.GaugeValue, ob[6]*100, r.SerialNumber),
			prometheus.MustNewConstMetric(tempest.Temperature, prometheus.GaugeValue, ob[7], r.SerialNumber),
			prometheus.MustNewConstMetric(tempest.Humidity, prometheus.GaugeValue, ob[8], r.SerialNumber),
			prometheus.MustNewConstMetric(tempest.Illuminance, prometheus.GaugeValue, ob[9], r.SerialNumber),
			prometheus.MustNewConstMetric(tempest.UV, prometheus.GaugeValue, ob[10], r.SerialNumber),
			prometheus.MustNewConstMetric(tempest.Irradiance, prometheus.GaugeValue, ob[11], r.SerialNumber),
			prometheus.MustNewConstMetric(tempest.RainRate, prometheus.GaugeValue, ob[12], r.SerialNumber),
		}
		// todo: lightning
		if len(ob) >= 17 {
			metrics = append(metrics,
				prometheus.MustNewConstMetric(tempest.Battery, prometheus.GaugeValue, ob[16], r.SerialNumber),
			)
		}
		if len(ob) >= 18 {
			metrics = append(metrics,
				prometheus.MustNewConstMetric(tempest.ReportInterval, prometheus.GaugeValue, ob[17]*60, r.SerialNumber),
			)
		}

		out = append(out, withTime(int64(ob[0]), metrics)...)
	}

	return out
}

type deviceStatusReport struct {
	SerialNumber string `json:"serial_number"`

	// "device_status"
	Type string `json:"type"`

	HubSn            string  `json:"hub_sn"`
	Timestamp        int     `json:"timestamp"`
	Uptime           int     `json:"uptime"`
	Voltage          float64 `json:"voltage"`
	FirmwareRevision int     `json:"firmware_revision"`
	Rssi             int     `json:"rssi"`
	HubRssi          int     `json:"hub_rssi"`

	// Binary Value	Applies to device	Status description
	// 0b000000000	All	Sensors OK
	// 0b000000001	AIR, Tempest	lightning failed
	// 0b000000010	AIR, Tempest	lightning noise
	// 0b000000100	AIR, Tempest	lightning disturber
	// 0b000001000	AIR, Tempest	pressure failed
	// 0b000010000	AIR, Tempest	temperature failed
	// 0b000100000	AIR, Tempest	rh failed
	// 0b001000000	SKY, Tempest	wind failed
	// 0b010000000	SKY, Tempest	precip failed
	// 0b100000000	SKY, Tempest	light/uv failed
	// any bits above 0b100000000 are reserved for internal use and should be ignored
	SensorStatus int `json:"sensor_status"`

	// 0	Debugging is disabled
	// 1	Debugging is enabled
	Debug int `json:"debug"`
}

func (r deviceStatusReport) Metrics() []prometheus.Metric {
	return nil
}

type hubStatusReport struct {
	SerialNumber string `json:"serial_number"`

	// "hub_status"
	Type string `json:"type"`

	FirmwareRevision string  `json:"firmware_revision"`
	Uptime           float64 `json:"uptime"`
	Rssi             float64 `json:"rssi"`
	Timestamp        int64   `json:"timestamp"`

	// BOR	Brownout reset
	// PIN	PIN reset
	// POR	Power reset
	// SFT	Software reset
	// WDG	Watchdog reset
	// WWD	Window watchdog reset
	// LPW	Low-power reset
	ResetFlags string `json:"reset_flags"`

	Seq int   `json:"seq"`
	Fs  []int `json:"fs"`

	// 0	Version
	// 1	Reboot Count
	// 2	I2C Bus Error Count
	// 3	Radio Status (0 = Radio Off, 1 = Radio On, 3 = Radio Active)
	// 4	Radio Network ID
	RadioStats []float64 `json:"radio_stats"`

	MqttStats []int `json:"mqtt_stats"`
}

func (r hubStatusReport) Metrics() []prometheus.Metric {
	return withTime(r.Timestamp, []prometheus.Metric{
		prometheus.MustNewConstMetric(tempest.Uptime, prometheus.CounterValue, r.Uptime, r.SerialNumber),
		prometheus.MustNewConstMetric(tempest.Rssi, prometheus.GaugeValue, r.Rssi, r.SerialNumber),
		prometheus.MustNewConstMetric(tempest.Reboots, prometheus.CounterValue, r.RadioStats[1], r.SerialNumber),
		prometheus.MustNewConstMetric(tempest.BusErrors, prometheus.CounterValue, r.RadioStats[2], r.SerialNumber),
	})
}

func withTime(unix int64, metrics []prometheus.Metric) []prometheus.Metric {
	t := time.Unix(unix, 0)
	out := make([]prometheus.Metric, 0, len(metrics))
	for _, m := range metrics {
		out = append(out, prometheus.NewMetricWithTimestamp(t, m))
	}
	return out
}
