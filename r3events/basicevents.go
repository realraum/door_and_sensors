// (c) Bernhard Tittelbach, 2013

package r3events

type DoorLockUpdate struct {
	Locked bool
	Ts     int64
}

type DoorAjarUpdate struct {
	Shut bool
	Ts   int64
}

type BackdoorAjarUpdate struct {
	Shut bool
	Ts   int64
}

type DoorCommandEvent struct {
	Command string
	Using   string
	Who     string
	Ts      int64
}

type DoorManualMovementEvent struct {
	Ts int64
}

type DoorProblemEvent struct {
	Severity int
	Problem  string
	Ts       int64
}

type BoreDoomButtonPressEvent struct {
	Ts int64
}

type TempSensorUpdate struct {
	Location string
	Value    float64
	Ts       int64
}

type TempOverThreshold struct {
	Location  string
	Value     float64
	Threshold float64
	Ts        int64
}

type IlluminationSensorUpdate struct {
	Location string
	Value    int64
	Ts       int64
}

type DustSensorUpdate struct {
	Location string
	Value    int64
	Ts       int64
}

type RelativeHumiditySensorUpdate struct {
	Location string
	Percent  float64
	Ts       int64
}

type PressureUpdate struct {
	Location string
	HPa      float64
	Ts       int64
}

type Voltage struct {
	Location string
	Value    float64
	Min      float64 //optional
	Max      float64 //optional
	Percent  float64 //optional
	Ts       int64
}

type UPSPowerUpdate struct {
	OnBattery      bool
	PercentBattery float64
	LineVoltage    float64
	LoadPercent    float64
	Ts             int64
}

type NetDHCPACK struct {
	Mac  string
	IP   string
	Name string
	Ts   int64
}

type NetGWStatUpdate struct {
	WifiRX     int32
	WifiTX     int32
	EthernetRX int32
	EthernetTX int32
	InternetRX int32
	InternetTX int32
	NumNeigh   int32
	Ts         int64
}

type GasLeakAlert struct {
	Ts int64
}

type TimeTick struct {
	Ts int64
}

type MovementSensorUpdate struct {
	Sensorindex int
	Ts          int64
}

type LaserCutter struct {
	IsHot bool
	Who   string
	Ts    int64
}

type FoodOrderRequest struct {
	Who        string
	Preference string
	Ts         int64
}

type FoodOrderInvite struct {
	Who   string
	Where string
	URL   string
	Ts    int64
}

type FoodOrderETA struct {
	TSofInvite int64
	ETA        int64
	Ts         int64
}

type VentilationState struct {
	Damper1 string
	Damper2 string
	Damper3 string
	Fan     string
	Ts      int64
}

type SonOffSensorBMP280 struct {
	Temperature float64
	Pressure    float64
}

type SonOffSensor struct {
	Time     string
	TempUnit string
	BMP280   SonOffSensorBMP280
}
