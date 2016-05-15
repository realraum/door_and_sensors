// (c) Bernhard Tittelbach, 2015

package main

import (
	"container/heap"
	"time"

	"git.eclipse.org/gitroot/paho/org.eclipse.paho.mqtt.golang.git"
	"github.com/btittelbach/astrotime"
	"github.com/realraum/door_and_sensors/r3events"
)

const LATITUDE = float64(47.065554)
const LONGITUDE = float64(15.450435)

type upcomingEvent struct {
	name         string
	havesunlight bool
	evt_time     time.Time
}

type eventHeap []upcomingEvent //propably less overhead than manageing and garbage collecting all those pointers if we did []*upcomingEvent

//--- Sort Interface for eventHeap ----
func (h eventHeap) Len() int {
	return len(h)
}
func (h eventHeap) Less(i, j int) bool {
	return h[i].evt_time.Before(h[j].evt_time)
}
func (h eventHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

//---- container/heap Interface for eventHeap ----
func (h *eventHeap) Push(x interface{}) {
	*h = append(*h, x.(upcomingEvent))
}

func (h *eventHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[0 : n-1]
	return item
}

func (h *eventHeap) Peek() *upcomingEvent {
	if h.Len() > 0 {
		return &((*h)[0])
	} else {
		return nil
	}
}

func calcNextSolarElevationEvent(now time.Time) upcomingEvent {
	eventheap := new(eventHeap)
	heap.Init(eventheap)
	heap.Push(eventheap, upcomingEvent{"AstronomicalDawn", false, astrotime.NextDawn(now, LATITUDE, LONGITUDE, astrotime.ASTRONOMICAL_DAWN)})
	heap.Push(eventheap, upcomingEvent{"NauticalDawn", false, astrotime.NextDawn(now, LATITUDE, LONGITUDE, astrotime.NAUTICAL_DAWN)})
	heap.Push(eventheap, upcomingEvent{"CivilDawn", false, astrotime.NextDawn(now, LATITUDE, LONGITUDE, astrotime.CIVIL_DAWN)})
	heap.Push(eventheap, upcomingEvent{"Sunrise", false, astrotime.NextDawn(now, LATITUDE, LONGITUDE, astrotime.SUNRISE)})
	heap.Push(eventheap, upcomingEvent{"GoldenHour", true, astrotime.NextDawn(now, LATITUDE, LONGITUDE, astrotime.GOLDEN_HOUR)})
	heap.Push(eventheap, upcomingEvent{"CityIndoorLights", true, astrotime.NextDawn(now, LATITUDE, LONGITUDE, astrotime.CITY_INDOOR_LIGHTS)})
	heap.Push(eventheap, upcomingEvent{"CityIndoorLights", false, astrotime.NextDusk(now, LATITUDE, LONGITUDE, astrotime.CITY_INDOOR_LIGHTS)})
	heap.Push(eventheap, upcomingEvent{"GoldenHour", false, astrotime.NextDusk(now, LATITUDE, LONGITUDE, astrotime.GOLDEN_HOUR)})
	heap.Push(eventheap, upcomingEvent{"Sunset", false, astrotime.NextDusk(now, LATITUDE, LONGITUDE, astrotime.SUNSET)})
	heap.Push(eventheap, upcomingEvent{"CivilDusk", false, astrotime.NextDusk(now, LATITUDE, LONGITUDE, astrotime.CIVIL_DUSK)})
	heap.Push(eventheap, upcomingEvent{"NauticalDusk", false, astrotime.NextDusk(now, LATITUDE, LONGITUDE, astrotime.NAUTICAL_DUSK)})
	heap.Push(eventheap, upcomingEvent{"AstronomicalDusk", false, astrotime.NextDusk(now, LATITUDE, LONGITUDE, astrotime.ASTRONOMICAL_DUSK)})
	return *(eventheap.Peek())
}

func MetaEventRoutine_DuskDawnEventGenerator(mqttc *mqtt.Client) {
	for {
		now := time.Now()
		upcoming_event := calcNextSolarElevationEvent(now)
		//block until time of next event has passed (add 30s, otherwise due to rounding in astrotime we might send event twice)
		<-time.NewTimer(upcoming_event.evt_time.Sub(now) + time.Minute).C
		mqttc.Publish(r3events.TOPIC_META_DUSKORDAWN, MQTT_QOS_4STPHANDSHAKE, true, r3events.MarshalEvent2ByteOrPanic(r3events.DuskOrDawn{Event: upcoming_event.name, HaveSunlight: upcoming_event.havesunlight, Ts: upcoming_event.evt_time.Unix()}))
	}
}
