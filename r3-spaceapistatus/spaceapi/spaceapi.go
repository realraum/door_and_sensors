// (c) Bernhard Tittelbach, 2013

// spaceapi.go
package spaceapi

import (
	"encoding/json"
	"time"
)

type SpaceInfo map[string]interface{}

type SpaceDoorLockSensor struct {
	value       bool
	location    string
	name        string
	description string
}

type SpaceDoorAjarSensor struct {
	value       bool
	location    string
	name        string
	description string
}

type SpaceEvent struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Timestamp  int64  `json:"timestamp"`
	Extra      string `json:"extra"`
	validuntil time.Time
}

type SpaceStateIcon struct {
	OpenIconURI  string `json:"open"`
	CloseIconURI string `json:"closed"`
}

type SpaceState struct {
	Open          []bool          `json:"open"`
	LastChange    int64           `json:"lastchange",omitempty`
	TriggerPerson string          `json:"trigger_person",omitempty`
	Message       string          `json:"message",omitempty`
	Icon          *SpaceStateIcon `json:"icon",omitempty`
}

type SpaceLocation struct {
	Address string  `json:"address",omitempty`
	Lat     float64 `json:"lat",omitempty`
	Lon     float64 `json:"lon",omitempty`
}

func MakeTempSensor(name, where, unit string, value float64, timestamp int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": "",
		"timestamp":   timestamp}
	return SpaceInfo{"temperature": listofwhats}
}

func MakeTempCSensor(name, where string, value float64, timestamp int64) SpaceInfo {
	return MakeTempSensor(name, where, "\u00b0C", value, timestamp)
}

func MakeIlluminationSensor(name, where, unit string, value int64, timestamp int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": "",
		"timestamp":   timestamp}
	return SpaceInfo{"ext_illumination": listofwhats}
}

func MakeHumiditySensor(name, where, unit string, value float64, timestamp int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": "relative humidity level",
		"timestamp":   timestamp}
	return SpaceInfo{"humidity": listofwhats}
}

func MakePowerConsumptionSensor(name, where, unit string, value, timestamp int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": "",
		"timestamp":   timestamp}
	return SpaceInfo{"power_consumption": listofwhats}
}

func MakeNetworkConnectionsSensor(name, where, nettype string, value, machines, timestamp int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"type":        nettype,
		"machines":    machines,
		"location":    where,
		"name":        name,
		"description": "",
		"timestamp":   timestamp}
	return SpaceInfo{"network_connections": listofwhats}
}

func MakeMemberCountSensor(name, where string, value int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"location":    where,
		"name":        name,
		"description": ""}
	return SpaceInfo{"total_member_count": listofwhats}
}

func MakeDoorLockSensor(name, where string, value bool) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"location":    where,
		"name":        name,
		"description": ""}
	return SpaceInfo{"door_locked": listofwhats}
}

func MakeDoorAjarSensor(name, where string, value bool) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"location":    where,
		"name":        name,
		"description": ""}
	return SpaceInfo{"ext_door_ajar": listofwhats}
}

func MakeLasercutterHotSensor(name, where string, value bool) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"location":    where,
		"name":        name,
		"description": "Indicates if the lasercutter is in use"}
	return SpaceInfo{"ext_lasercutter_hot": listofwhats}
}

func MakeVoltageSensor(name, where, unit string, value float64, timestamp int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": "Voltage",
		"timestamp":   timestamp}
	return SpaceInfo{"ext_voltage": listofwhats}
}

func MakeBatteryChargeSensor(name, where, unit string, percentcharge float64, timestamp int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       percentcharge,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": "Charge of battery",
		"timestamp":   timestamp}
	return SpaceInfo{"ext_batterycharge": listofwhats}
}

func (nsi SpaceInfo) MergeInSensor(sensorinfo SpaceInfo) {
	if nsi["sensors"] == nil {
		nsi["sensors"] = SpaceInfo{}
		//~ listofwhats := make([]SpaceInfo, 1)
		//~ listofwhats[0] = sensortype.(SpaceInfo)
		//~ sensorobj := SpaceInfo{what: listofwhats}
		//~ nsi["sensors"] = sensorobj
	}
	sensorobj := nsi["sensors"].(SpaceInfo)
	for what, subsensorobjlist := range sensorinfo {
		if sensorobj[what] == nil {
			sensorobj[what] = subsensorobjlist
		} else {
			existingsensorobjslist := sensorobj[what].([]SpaceInfo)
			for _, newsensorobj := range subsensorobjlist.([]SpaceInfo) {
				foundandsubstituted := false
				for i := 0; i < len(existingsensorobjslist); i++ {
					if existingsensorobjslist[i]["name"] == newsensorobj["name"] {
						existingsensorobjslist[i] = newsensorobj
						foundandsubstituted = true
					}
				}
				if foundandsubstituted == false {
					sensorobj[what] = append(sensorobj[what].([]SpaceInfo), newsensorobj)
					//note that we do not change existingsensorobjslist here but directly sensorobj[what] !!
					//the implications being that, if we have several newsensorobj in the list:
					//  a) optimisation: we only check them against the existing ones and spare ourselves the work of checking a newsensorobj's name against a just added other newsensorobjs's name
					//  b) if the array sensorinfo[what] has several objects with the same name, nsi["sensors"] will also end up with these name conflicts
				}
			}
		}
	}
}

func (nsi SpaceInfo) CleanOutdatedSensorData(older_than time.Duration) SpaceInfo {
	now := time.Now()
	if nsi["sensors"] != nil {
		sensorobj := nsi["sensors"].(SpaceInfo)
		for sensortype, subsensorobjlist := range sensorobj {
			if subsensorobjlist != nil {
				existingsensorobjslist := subsensorobjlist.([]SpaceInfo)
				for i := len(existingsensorobjslist) - 1; i >= 0; i-- {
					tsi, inmap := existingsensorobjslist[i]["timestamp"]
					if !inmap {
						continue
					}
					ts, taok := tsi.(int64)
					if !taok {
						continue
					}
					if now.Sub(time.Unix(ts, 0)) > older_than {
						var newobjlist []SpaceInfo
						if i > 0 {
							newobjlist = existingsensorobjslist[0:i]
						} else {
							newobjlist = make([]SpaceInfo, 0)
						}
						if i < len(existingsensorobjslist)-1 {
							newobjlist = append(newobjlist, existingsensorobjslist[i+1:len(existingsensorobjslist)]...)
						}
						existingsensorobjslist = newobjlist
					}
				}
				if len(existingsensorobjslist) == 0 {
					delete(sensorobj, sensortype)
				} else {
					sensorobj[sensortype] = existingsensorobjslist
				}
			}
		}
	}
	return nsi
}

func (nsi SpaceInfo) AddProjectsURLs(projecturls []string) SpaceInfo {
	if nsi["projects"] == nil {
		nsi["projects"] = projecturls
		//~ listofwhats := make([]SpaceInfo, 1)
		//~ listofwhats[0] = sensortype.(SpaceInfo)
		//~ sensorobj := SpaceInfo{what: listofwhats}
		//~ nsi["sensors"] = sensorobj
	}
	return nsi
}

func (nsi SpaceInfo) AddBaseExt(key, value string) SpaceInfo {
	if nsi[key] == nil {
		nsi[key] = value
	}
	return nsi
}

func (nsi SpaceInfo) SetSpaceContactInfo(channel, info string, is_issue_report_channel bool) SpaceInfo {
	if nsi["contact"] == nil {
		nsi["contact"] = make([]SpaceInfo, 1)
	}
	contact := nsi["contact"].(SpaceInfo)
	contact[channel] = info
	if is_issue_report_channel && (channel == "email" || channel == "issue_mail" || channel == "twitter" || channel == "ml") {
		if nsi["issue_report_channels"] == nil {
			nsi["issue_report_channels"] = []string{channel}
		} else {
			nsi["issue_report_channels"] = append(nsi["issue_report_channels"].([]string), channel)
		}
	}
	return nsi
}

func (nsi SpaceInfo) SetSpaceContactPhone(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("phone", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactSip(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("sip", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactIRC(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("irc", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactTwitter(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("twitter", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactFaceBook(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("facebook", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactGooglePlus(info string) SpaceInfo {
	if nsi["contact"] == nil {
		nsi["contact"] = make([]SpaceInfo, 1)
	}
	contact := nsi["contact"].(SpaceInfo)
	contact["google"] = SpaceInfo{"plus": info}
	return nsi
}

func (nsi SpaceInfo) SetSpaceContactIdentica(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("identica", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactFourSquare(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("foursquare", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactEmail(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("email", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactMailinglist(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("ml", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactJabber(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("jabber", info, is_issue_report_channel)
}

func (nsi SpaceInfo) SetSpaceContactIssueMail(info string, is_issue_report_channel bool) SpaceInfo {
	return nsi.SetSpaceContactInfo("issue_mail", info, is_issue_report_channel)
}

func (nsi SpaceInfo) AddSpaceFeed(feedtype, url, typestr string) SpaceInfo {
	newfeed := SpaceInfo{"url": url, "type": typestr}
	if nsi["feeds"] == nil {
		nsi["feeds"] = SpaceInfo{feedtype: newfeed}
	} else {
		feedobj, ok := nsi["feeds"].(SpaceInfo) //type assertion (panics if false)
		if ok {
			feedobj[feedtype] = newfeed
		} else {
			panic("Wrong Type of feedobj: Should never happen")
		}
	}
	return nsi
}

func (nsi SpaceInfo) AddSpaceEvent(name, eventtype, extra string, unixts int64, validity_duration time.Duration) SpaceInfo {
	newevent := SpaceEvent{name, eventtype, unixts, extra, time.Now().Add(validity_duration)}
	if nsi["events"] == nil {
		eventlist := make([]SpaceEvent, 1)
		eventlist[0] = newevent
		nsi["events"] = eventlist
	} else {
		eventlist, ok := nsi["events"].([]SpaceEvent) //type assertion
		if ok {
			nsi["events"] = append(eventlist, newevent)
		} else {
			panic("Wrong Type of eventlist: Should never happen")
		}
	}
	return nsi
}

func (nsi SpaceInfo) CleanOutdatedSpaceEvents() SpaceInfo {
	if nsi["events"] == nil {
		return nsi
	}
	events := nsi["events"].([]SpaceEvent)
	now := time.Now()
	for idx := len(events) - 1; idx >= 0; idx-- {
		if events[idx].validuntil.Before(now) {
			//delete element, no need to preserve order
			events[idx] = events[len(events)-1]
			// events[len(events)-1] = nil
			events = events[:len(events)-1]
		}
	}
	if len(events) > 0 {
		nsi["events"] = events
	} else {
		delete(nsi, "events")
	}
	return nsi
}

func (nsi SpaceInfo) SetSpaceLocation(l SpaceLocation) SpaceInfo {
	nsi["location"] = l
	return nsi
}

func (nsi SpaceInfo) SetSpaceState(state SpaceState) SpaceInfo {
	state.LastChange = time.Now().Unix()
	nsi["state"] = state
	//add fields outside for 0.12 backward compatibility
	if state.Open != nil && len(state.Open) > 0 {
		nsi["open"] = state.Open[0]
	} else {
		delete(nsi, "open")
	}
	if state.LastChange != 0 {
		nsi["lastchange"] = state.LastChange
	} else {
		delete(nsi, "lastchange")
	}
	if len(state.Message) > 0 {
		nsi["message"] = state.Message
	} else {
		delete(nsi, "message")
	}
	if state.Icon != nil {
		nsi["icon"] = state.Icon
	} else {
		delete(nsi, "icon")
	}
	return nsi
}

func (nsi SpaceInfo) UpdateSpaceStatus(open bool, status string) SpaceInfo {
	state, ok := nsi["state"].(SpaceState)
	if ok {
		state.Message = status
		state.Open = []bool{open}
	}
	return nsi.SetSpaceState(state)
}

func (nsi SpaceInfo) UpdateSpaceStatusNotKnown(status string) SpaceInfo {
	state, ok := nsi["state"].(SpaceState)
	if ok {
		state.Message = status
		state.Open = []bool{}
	}
	return nsi.SetSpaceState(state)
}

func NewSpaceInfo(space string, url string, logo string) SpaceInfo {
	nsi := map[string]interface{}{
		"api":     "0.13",
		"space":   space,
		"url":     url,
		"logo":    logo,
		"contact": SpaceInfo{},
	}
	return nsi
}

func (data SpaceInfo) MakeJSON() ([]byte, error) {
	msg, err := json.Marshal(data)
	if err == nil {
		return msg, nil
	}
	return nil, err
}
