package main

import (
	"testing"
	"time"
)

func TestNextEvents(t *testing.T) {

	evt := calcNextSolarElevationEvent(time.Date(2016, 5, 15, 3, 00, 0, 0, time.UTC))
	if evt.name != "Sunrise" {
		t.Fatalf("Unexpected next event: %+v", evt)
	}
	evt = calcNextSolarElevationEvent(time.Date(2016, 5, 15, 4, 00, 0, 0, time.UTC))
	if evt.name != "GoldenHour" {
		t.Fatalf("Unexpected next event: %+v", evt)
	}
	evt = calcNextSolarElevationEvent(time.Date(2016, 5, 15, 5, 00, 0, 0, time.UTC))
	if evt.name != "CityIndoorLights" && evt.havesunlight != true {
		t.Fatalf("Unexpected next event: %+v", evt)
	}
	evt = calcNextSolarElevationEvent(time.Date(2016, 5, 15, 12, 00, 0, 0, time.UTC))
	if evt.name != "CityIndoorLights" && evt.havesunlight != false {
		t.Fatalf("Unexpected next event: %+v", evt)
	}
	evt = calcNextSolarElevationEvent(time.Date(2016, 5, 15, 18, 00, 0, 0, time.UTC))
	if evt.name != "Sunset" {
		t.Fatalf("Unexpected next event: %+v", evt)
	}
	evt = calcNextSolarElevationEvent(time.Date(2016, 5, 15, 19, 00, 0, 0, time.UTC))
	if evt.name != "CivilDusk" {
		t.Fatalf("Unexpected next event: %+v", evt)
	}
	evt = calcNextSolarElevationEvent(time.Date(2016, 5, 15, 20, 00, 0, 0, time.UTC))
	if evt.name != "AstronomicalDusk" {
		t.Fatalf("Unexpected next event: %+v", evt)
	}
}
