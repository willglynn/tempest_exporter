package tempest

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	Uptime    *prometheus.Desc
	Rssi      *prometheus.Desc
	Reboots   *prometheus.Desc
	BusErrors *prometheus.Desc

	Illuminance    *prometheus.Desc
	UV             *prometheus.Desc
	RainRate       *prometheus.Desc
	Wind           *prometheus.Desc // "lull", "avg", "gust", "rapid"
	WindDirection  *prometheus.Desc
	Battery        *prometheus.Desc
	ReportInterval *prometheus.Desc
	Irradiance     *prometheus.Desc
	RainTotal      *prometheus.Desc
	Pressure       *prometheus.Desc
	Temperature    *prometheus.Desc // "air", "wetbulb"
	Humidity       *prometheus.Desc
)

var All []*prometheus.Desc

func init() {
	Uptime = prometheus.NewDesc("tempest_uptime_seconds_total", "The uptime of the device", []string{"instance"}, nil)
	Rssi = prometheus.NewDesc("tempest_rssi_dbm", "A measurement of wireless signal strength", []string{"instance"}, nil)
	Reboots = prometheus.NewDesc("tempest_reboots_total", "The number of times the device has rebooted", []string{"instance"}, nil)
	BusErrors = prometheus.NewDesc("tempest_bus_errors_total", "The number of I2C bus errors experienced by the device", []string{"instance"}, nil)

	Illuminance = prometheus.NewDesc("tempest_illuminance_lux", "A measurement of luminous flux per unit area", []string{"instance"}, nil)
	UV = prometheus.NewDesc("tempest_uv_index", "A measurement of ultraviolet light intensity", []string{"instance"}, nil)
	RainRate = prometheus.NewDesc("tempest_rain_rate_mm_min", "The amount of rain which fell on the sensor in the previous minute", []string{"instance"}, nil)
	Wind = prometheus.NewDesc("tempest_wind_ms", "A wind speed measurement", []string{"instance", "kind"}, nil)
	WindDirection = prometheus.NewDesc("tempest_wind_direction_degrees", "The direction from which the wind is blowing", []string{"instance"}, nil)
	Battery = prometheus.NewDesc("tempest_battery_volts", "The electric potential of the battery", []string{"instance"}, nil)
	ReportInterval = prometheus.NewDesc("tempest_report_interval_s", "The interval over with which the station makes reports", []string{"instance"}, nil)
	Irradiance = prometheus.NewDesc("tempest_irradiance_w_m2", "The total solar irradiance, expressed in watts per square meter", []string{"instance"}, nil)
	RainTotal = prometheus.NewDesc("tempest_rainfall_total", "The amount of accumulated rain", []string{"instance"}, nil)
	Pressure = prometheus.NewDesc("tempest_pressure_pa", "A barometric pressure measurement", []string{"instance"}, nil)
	Temperature = prometheus.NewDesc("tempest_temperature_c", "A temperature measurement", []string{"instance", "kind"}, nil)
	Humidity = prometheus.NewDesc("tempest_humidity_percent", "A relative humidity measurement", []string{"instance"}, nil)

	// todo: lightning

	All = []*prometheus.Desc{
		Uptime,
		Rssi,
		Reboots,
		BusErrors,

		Illuminance,
		UV,
		RainRate,
		Wind,
		WindDirection,
		Battery,
		ReportInterval,
		Irradiance,
		RainTotal,
		Pressure,
		Temperature,
		Humidity,
	}
}
