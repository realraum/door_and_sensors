package spaceapi

import (
	"testing"
	"time"
)

func TestCleanSensors(t *testing.T) {
	spaceapidata := NewSpaceInfo("realraum", "http://realraum.at", "http://realraum.at/logo-red_250x250.png", "http://realraum.at/logo-re_open_100x100.png", "http://realraum.at/logo-re_empty_100x100.png", 47.065554, 15.450435).AddSpaceAddress("Brockmanngasse 15, 8010 Graz, Austria")
	spaceapidata.MergeInSensor(MakeTempCSensor("Temp1", "Here", 20.2, time.Now().Add(-3*time.Minute).Unix()))
	spaceapidata.MergeInSensor(MakeTempCSensor("Temp2", "There", 19.8, time.Now().Add(-1*time.Minute).Unix()))
	temps := spaceapidata["sensors"].(SpaceInfo)["temperature"].([]SpaceInfo)
	t.Logf("%+v", temps)
	if len(temps) != 2 {
		t.Fatal("Did not add temp sensors")
	}
	spaceapidata.CleanOutdatedSensorData(2 * time.Minute)
	temps = spaceapidata["sensors"].(SpaceInfo)["temperature"].([]SpaceInfo)
	t.Logf("%+v", temps)
	if len(temps) != 1 {
		t.Fatalf("Failed to remove Sensor, num sensors remaining: %d !! \n%+v", len(temps), temps)
	}
	spaceapidata.CleanOutdatedSensorData(2 * time.Second)
	temps2, inmap := spaceapidata["sensors"].(SpaceInfo)["temperature"]
	if inmap {
		temps = temps2.([]SpaceInfo)
		t.Fatalf("Failed to remove last Sensor, num sensors remaining: %d !!! \n%+v", len(temps), temps)
	}
}
