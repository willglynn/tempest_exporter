package tempestudp

import (
	"fmt"
	"math"
	"testing"
)

func Test_wetBulbTemperatureC(t *testing.T) {
	type args struct {
		temperatureC       float64
		humidityPercent    float64
		stationPressureHpa float64
	}
	tests := []struct {
		args args
		want float64
	}{
		{args{25, 50, 900}, 17.71},
		{args{25, 90, 900}, 23.7},
		{args{30, 33, 1050}, 18.92},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("test %d", i), func(t *testing.T) {
			got := wetBulbTemperatureC(tt.args.temperatureC, tt.args.humidityPercent, tt.args.stationPressureHpa)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("wetBulbTemperatureC(%v, %v, %v) = %0.2f, want %v", tt.args.temperatureC, tt.args.humidityPercent, tt.args.stationPressureHpa, got, tt.want)
			}
		})
	}
}
