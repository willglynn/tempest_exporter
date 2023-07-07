package tempestudp

import (
	"math"
)

func saturatedVaporPressure(temperatureC float64) float64 {
	return 6.112 * math.Exp((17.67*temperatureC)/(243.5+temperatureC))
}

func wetBulbVaporPressure(tC, twC float64, stationPressureHpa float64) float64 {
	return saturatedVaporPressure(twC) - stationPressureHpa*(tC-twC)*0.00066*(1+0.00115*twC)
}

func wetBulbTemperatureC(temperatureC float64, humidityPercent float64, stationPressureHpa float64) float64 {
	// Determine the saturated vapor pressure for the station temperature
	eSaturated := saturatedVaporPressure(temperatureC)

	// Relative humidity is relative to full saturation, which gives us the actual vapor pressure at the station
	eHumidity := eSaturated * (humidityPercent / 100)

	// We can determine the vapor pressure for a given wet bulb temperature, but we can't readily invert that
	// Guess at a wet bulb temperature, adjusting up or down, until the vapor pressure for the wet bulb temperature
	// matches the vapor pressure implied by the relative humidity
	step := 8.0
	wetBulbC := temperatureC - step*2
	for i := 0; i < 10000; i++ {
		eWetBulb := wetBulbVaporPressure(temperatureC, wetBulbC, stationPressureHpa)

		delta := eHumidity - eWetBulb
		if math.Abs(delta) < 0.001 {
			// We converged
			return wetBulbC
		}

		// Are we stepping the right way?
		if math.Signbit(delta) == math.Signbit(step) {
			// Step again
		} else {
			// We overshot
			// Step the other direction, and take smaller steps
			step *= -0.25
		}
		wetBulbC += step
	}
	panic("failed to converge")
}
