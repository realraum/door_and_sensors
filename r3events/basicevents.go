// (c) Bernhard Tittelbach, 2013

package r3events

type DoorLockUpdate struct {
	Locked bool
	Ts     int64 `json:",omitempty"`
}

type DoorAjarUpdate struct {
	Shut bool
	Ts   int64 `json:",omitempty"`
}

type BackdoorAjarUpdate struct {
	Shut bool
	Ts   int64 `json:",omitempty"`
}

type DoorCommandEvent struct {
	Command string
	Using   string
	Who     string
	Ts      int64 `json:",omitempty"`
}

type DoorManualMovementEvent struct {
	Ts int64 `json:",omitempty"`
}

type DoorProblemEvent struct {
	Severity int
	Problem  string
	Ts       int64 `json:",omitempty"`
}

type BoreDoomButtonPressEvent struct {
	Ts int64 `json:",omitempty"`
}

type TempSensorUpdate struct {
	Location string
	Value    float64
	Ts       int64 `json:",omitempty"`
}

type TempOverThreshold struct {
	Location  string
	Value     float64
	Threshold float64
	Ts        int64 `json:",omitempty"`
}

type IlluminationSensorUpdate struct {
	Location string
	Value    int64
	Ts       int64 `json:",omitempty"`
}

type DustSensorUpdate struct {
	Location string
	Value    int64
	Ts       int64 `json:",omitempty"`
}

type RelativeHumiditySensorUpdate struct {
	Location string
	Percent  float64
	Ts       int64 `json:",omitempty"`
}

type BarometerUpdate struct {
	Location string
	HPa      float64
	Ts       int64 `json:",omitempty"`
}

type Voltage struct {
	Location string
	Value    float64
	Min      float64 //optional
	Max      float64 //optional
	Percent  float64 //optional
	Ts       int64   `json:",omitempty"`
}

type UPSPowerUpdate struct {
	OnBattery      bool
	PercentBattery float64
	LineVoltage    float64
	LoadPercent    float64
	Ts             int64 `json:",omitempty"`
}

type NetDHCPACK struct {
	Mac  string
	IP   string
	Name string
	Ts   int64 `json:",omitempty"`
}

type NetGWStatUpdate struct {
	WifiRX     int32
	WifiTX     int32
	EthernetRX int32
	EthernetTX int32
	InternetRX int32
	InternetTX int32
	NumNeigh   int32
	Ts         int64 `json:",omitempty"`
}

type GasLeakAlert struct {
	Ts int64 `json:",omitempty"`
}

type TimeTick struct {
	Ts int64 `json:",omitempty"`
}

type MovementSensorUpdate struct {
	Sensorindex int
	Ts          int64 `json:",omitempty"`
}

type LaserCutter struct {
	IsHot bool
	Who   string
	Ts    int64 `json:",omitempty"`
}

type FoodOrderRequest struct {
	Who        string
	Preference string
	Ts         int64 `json:",omitempty"`
}

type FoodOrderInvite struct {
	Who   string
	Where string
	URL   string
	Ts    int64 `json:",omitempty"`
}

type FoodOrderETA struct {
	TSofInvite int64
	ETA        int64
	Ts         int64 `json:",omitempty"`
}

type VentilationState struct {
	Damper1   string
	Damper2   string
	Damper3   string
	Fan       string
	OLGALock  bool
	LaserLock bool
	Ts        int64 `json:",omitempty"`
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
