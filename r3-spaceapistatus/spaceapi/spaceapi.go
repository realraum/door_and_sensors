// (c) Bernhard Tittelbach, 2013

// spaceapi.go
package spaceapi

import (
	"encoding/json"
	"time"
)

const max_num_events int = 4

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

func MakeTempSensor(name, where, unit string, value float64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": ""}
	return SpaceInfo{"temperature": listofwhats}
}

func MakeTempCSensor(name, where string, value float64) SpaceInfo {
	return MakeTempSensor(name, where, "\u00b0C", value)
}

func MakeIlluminationSensor(name, where, unit string, value int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": ""}
	return SpaceInfo{"ext_illumination": listofwhats}
}

func MakeHumiditySensor(name, where, unit string, value float64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": "relative humidity level"}
	return SpaceInfo{"humidity": listofwhats}
}

func MakePowerConsumptionSensor(name, where, unit string, value int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"unit":        unit,
		"location":    where,
		"name":        name,
		"description": ""}
	return SpaceInfo{"power_consumption": listofwhats}
}

func MakeNetworkConnectionsSensor(name, where, nettype string, value, machines int64) SpaceInfo {
	listofwhats := make([]SpaceInfo, 1)
	listofwhats[0] = SpaceInfo{
		"value":       value,
		"type":        nettype,
		"machines":    machines,
		"location":    where,
		"name":        name,
		"description": ""}
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

func (nsi SpaceInfo) AddSpaceFeed(feedtype, url string) SpaceInfo {
	newfeed := SpaceInfo{"url": url}
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

func (nsi SpaceInfo) AddSpaceEvent(name, eventtype, extra string) SpaceInfo {
	newevent := SpaceInfo{"name": name, "type": eventtype, "timestamp": time.Now().Unix(), "extra": extra}
	if nsi["events"] == nil {
		eventlist := make([]SpaceInfo, 1)
		eventlist[0] = newevent
		nsi["events"] = eventlist
	} else {
		eventlist, ok := nsi["events"].([]SpaceInfo) //type assertion
		if ok {
			if len(eventlist) >= max_num_events {
				eventlist = eventlist[1:]
			}
			nsi["events"] = append(eventlist, newevent)
		} else {
			panic("Wrong Type of eventlist: Should never happen")
		}
	}
	return nsi
}

func (nsi SpaceInfo) AddSpaceAddress(address string) SpaceInfo {
	nsi["address"] = address
	if nsi["location"] != nil {
		location, ok := nsi["location"].(SpaceInfo)
		if ok {
			location["address"] = address
		}
	}
	return nsi
}

func (nsi SpaceInfo) SetStatus(open bool, status string) {
	nsi["status"] = status
	nsi["open"] = open
	nsi["lastchange"] = time.Now().Unix()
	state, ok := nsi["state"].(SpaceInfo)
	if ok {
		state["message"] = status
		state["open"] = open
		state["lastchange"] = nsi["lastchange"]
	}
}

func NewSpaceInfo(space string, url string, logo string, open_icon string, closed_icon string, lat float64, lon float64) SpaceInfo {
	nsi := map[string]interface{}{
		"api":        "0.13",
		"space":      space,
		"url":        url,
		"logo":       logo,
		"open":       false,
		"lastchange": time.Now().Unix(),
		"icon": SpaceInfo{
			"open":   open_icon,
			"closed": closed_icon,
		},
		"state": SpaceInfo{
			"open":       false,
			"lastchange": time.Now().Unix(),
			"icon": SpaceInfo{
				"open":   open_icon,
				"closed": closed_icon},
		},
		"location": SpaceInfo{
			"lat": lat,
			"lon": lon},
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
