package tempestudp

import (
	"reflect"
	"strings"
	"testing"

	"tempest_exporter/tempest"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
)

func Test_rapidWindReport_Metrics(t *testing.T) {
	metricsTest(t, []metricsTestcase{
		{
			"typical",
			`{"serial_number":"ST-00019709","type":"rapid_wind","hub_sn":"HB-00031344","ob":[1688668572,0.85,113]}`,
			"ST-00019709", 1688668572,
			[]simpleMetric{
				{
					desc:   tempest.Wind,
					value:  0.85,
					labels: map[string]string{"kind": "rapid"},
				},
				{
					desc:  tempest.WindDirection,
					value: 113,
				},
			},
		},
	})
}

func Test_tempestObservationReport_Metrics(t *testing.T) {
	metricsTest(t, []metricsTestcase{
		{
			"typical",
			`{"serial_number":"ST-00019709","type":"obs_st","hub_sn":"HB-00031344","obs":[[1688668741,0.00,0.49,1.44,163,3,987.81,19.00,67.63,57687,4.38,480,0.000000,0,0,0,2.792,1]],"firmware_revision":156}`,
			"ST-00019709", 1688668741,
			[]simpleMetric{
				{
					desc:   tempest.Wind,
					value:  0,
					labels: map[string]string{"kind": "lull"},
				},
				{
					desc:   tempest.Wind,
					value:  0.49,
					labels: map[string]string{"kind": "avg"},
				},
				{
					desc:   tempest.Wind,
					value:  1.44,
					labels: map[string]string{"kind": "gust"},
				},
				{
					desc:  tempest.WindDirection,
					value: 163,
				},
				{
					desc:  tempest.Pressure,
					value: 98781,
				},
				{
					desc:  tempest.Temperature,
					value: 19.0,
				},
				{
					desc:  tempest.Humidity,
					value: 67.63,
				},
				{
					desc:  tempest.Illuminance,
					value: 57687,
				},
				{
					desc:  tempest.UV,
					value: 4.38,
				},
				{
					desc:  tempest.Irradiance,
					value: 480,
				},
				{
					desc:  tempest.RainRate,
					value: 0,
				},
				{
					desc:  tempest.Battery,
					value: 2.792,
				},
				{
					desc:  tempest.ReportInterval,
					value: 60, // seconds
				},
			},
		},
	})
}

func Test_hubStatusReport_Metrics(t *testing.T) {
	metricsTest(t, []metricsTestcase{
		{
			"typical",
			`{"serial_number":"HB-00031344","type":"hub_status","firmware_revision":"171","uptime":64275,"rssi":-44,"timestamp":1688666650,"reset_flags":"BOR,PIN,POR","seq":6419,"fs":[1,0,15675411,524288],"radio_stats":[25,1,0,3,16344],"mqtt_stats":[1,4]}`,
			"HB-00031344", 1688666650,
			[]simpleMetric{
				{
					desc:  tempest.Uptime,
					value: 64275,
				},
				{
					desc:  tempest.Rssi,
					value: -44,
				},
				{
					desc:  tempest.Reboots,
					value: 1,
				},
				{
					desc:  tempest.BusErrors,
					value: 0,
				},
			},
		},
	})
}

type simpleMetric struct {
	desc   *prometheus.Desc
	value  float64
	labels map[string]string
}

type metricsTestcase struct {
	name          string
	input         string
	wantInstance  string
	wantTimestamp int64
	wantMetrics   []simpleMetric
}

func metricsTest(t *testing.T, testcases []metricsTestcase) {
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := ParseReport([]byte(tc.input))
			if err != nil {
				t.Fatalf("error parsing input: %v", err)
			}

			got := report.Metrics()
			var gotSM []simpleMetric

			for _, gm := range got {
				var dm io_prometheus_client.Metric
				if err := gm.Write(&dm); err != nil {
					t.Fatal("unable to write metric", err)
				}

				var gotInstance string
				gotLabels := make(map[string]string)
				for _, label := range dm.GetLabel() {
					if label.GetName() == "instance" {
						gotInstance = label.GetValue()
					} else {
						gotLabels[label.GetName()] = label.GetValue()
					}
				}
				if gotInstance != tc.wantInstance {
					t.Errorf("instance = %q, want %q", gotInstance, tc.wantInstance)
				}
				if len(gotLabels) == 0 {
					gotLabels = nil
				}

				gotTimestamp := dm.GetTimestampMs() / 1000
				if gotTimestamp != tc.wantTimestamp {
					t.Errorf("timestamp = %v, want %v", gotTimestamp, tc.wantTimestamp)
				}

				var value float64
				if dm.GetCounter() != nil {
					value = dm.GetCounter().GetValue()
				} else if dm.GetGauge() != nil {
					value = dm.GetGauge().GetValue()
				}

				gotSM = append(gotSM, simpleMetric{
					desc:   gm.Desc(),
					value:  value,
					labels: gotLabels,
				})
			}

			var gotNames, wantNames []string
			for _, m := range gotSM {
				gotNames = append(gotNames, m.desc.String())
			}
			for _, m := range tc.wantMetrics {
				wantNames = append(wantNames, m.desc.String())
			}
			if !reflect.DeepEqual(gotNames, wantNames) {
				t.Errorf("metric descs: got\n  %s\n\nwant\n  %s", strings.Join(gotNames, "\n  "), strings.Join(wantNames, "\n  "))
				return
			}

			for i, gm := range gotSM {
				wm := tc.wantMetrics[i]
				if gm.value != wm.value {
					t.Errorf("%s\n  value = %v, want %v", gotNames[i], gm.value, wm.value)
				}
				if !reflect.DeepEqual(gm.labels, wm.labels) {
					t.Errorf("%s\n  labels = %#v, want %#v", gotNames[i], gm.labels, wm.labels)
				}
			}
		})
	}
}

func TestParseReport(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Report
		wantErr bool
	}{
		{name: "empty", wantErr: true},
		{
			name:  "rapid wind",
			input: `{"serial_number":"ST-00019709","type":"rapid_wind","hub_sn":"HB-00031344","ob":[1688666352,0.09,97]}`,
			want: &rapidWindReport{
				SerialNumber: "ST-00019709",
				Type:         "rapid_wind",
				HubSn:        "HB-00031344",
				Ob:           []float64{1688666352, 0.09, 97},
			},
		},
		{
			name:  "tempest observation",
			input: `{"serial_number":"ST-00019709","type":"obs_st","hub_sn":"HB-00031344","obs":[[1688666521,0.00,0.67,1.48,144,3,988.36,19.41,64.52,128807,9.41,1073,0.000000,0,0,0,2.792,1]],"firmware_revision":156}`,
			want: &TempestObservationReport{
				SerialNumber:     "ST-00019709",
				Type:             "obs_st",
				HubSn:            "HB-00031344",
				Obs:              [][]float64{{1688666521, 0.00, 0.67, 1.48, 144, 3, 988.36, 19.41, 64.52, 128807, 9.41, 1073, 0.000000, 0, 0, 0, 2.792, 1}},
				FirmwareRevision: 156,
			},
		},
		{
			name:  "device status",
			input: `{"serial_number":"ST-00019709","type":"device_status","hub_sn":"HB-00031344","timestamp":1688666521,"uptime":63807156,"voltage":2.792,"firmware_revision":156,"rssi":-82,"hub_rssi":-78,"sensor_status":0,"debug":0}`,
			want: &deviceStatusReport{
				SerialNumber:     "ST-00019709",
				Type:             "device_status",
				HubSn:            "HB-00031344",
				Timestamp:        1688666521,
				Uptime:           63807156,
				Voltage:          2.792,
				FirmwareRevision: 156,
				Rssi:             -82,
				HubRssi:          -78,
				SensorStatus:     0,
				Debug:            0,
			},
		},
		{
			name:  "hub status",
			input: `{"serial_number":"HB-00031344","type":"hub_status","firmware_revision":"171","uptime":63975,"rssi":-44,"timestamp":1688666350,"reset_flags":"BOR,PIN,POR","seq":6389,"fs":[1,0,15675411,524288],"radio_stats":[25,1,0,3,16344],"mqtt_stats":[1,4]}`,
			want: &hubStatusReport{
				SerialNumber:     "HB-00031344",
				Type:             "hub_status",
				FirmwareRevision: "171",
				Uptime:           63975,
				Rssi:             -44,
				Timestamp:        1688666350,
				ResetFlags:       "BOR,PIN,POR",
				Seq:              6389,
				Fs:               []int{1, 0, 15675411, 524288},
				RadioStats:       []float64{25, 1, 0, 3, 16344},
				MqttStats:        []int{1, 4},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReport([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReport() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseReport() got = %v, want %v", got, tt.want)
			}
		})
	}
}
