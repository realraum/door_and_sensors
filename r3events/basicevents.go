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
	Sensorindex int
	Value       float64
	Ts          int64
}

type IlluminationSensorUpdate struct {
	Sensorindex int
	Value       int64
	Ts          int64
}

type DustSensorUpdate struct {
	Sensorindex int
	Value       int64
	Ts          int64
}

type RelativeHumiditySensorUpdate struct {
	Sensorindex int
	Percent     int
	Ts          int64
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
	Ts    int64
}

type YamahaIRCommand struct {
	Cmd string
	Ts  int64
}

type SendRF433Code struct {
	Code string
	Ts   int64
}
