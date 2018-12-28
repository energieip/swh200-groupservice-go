package service

import (
	"strconv"
	"time"

	"github.com/energieip/common-group-go/pkg/groupmodel"
	"github.com/energieip/common-led-go/pkg/driverled"
	"github.com/energieip/common-sensor-go/pkg/driversensor"
	"github.com/romana/rlog"
)

const (
	EventChange = "change"
	EventStop   = "stop"
	EventManual = "manual"
)

// Group logical
type Group struct {
	Event           chan map[string]*groupmodel.GroupRuntime
	Runtime         groupmodel.GroupRuntime
	NewSetpoint     int
	Setpoint        int
	Brightness      int
	Presence        bool
	Slope           int
	TimeToAuto      int
	Scale           int    //brightness correction scale
	DbID            string //Database entry ID
	PresenceTimeout int
	Error           int
}

func (s *GroupService) updateDatabase(group Group, status groupmodel.GroupStatus) error {

	if group.DbID == "" {
		//Fetch existing group status
		criteria := make(map[string]interface{})
		criteria["Group"] = group.Runtime.Group
		groupStored, err := s.db.GetRecord(groupmodel.DbStatusName, groupmodel.TableStatusName, criteria)
		if err == nil && groupStored != nil {
			m := groupStored.(map[string]interface{})
			id, ok := m["id"]
			if !ok {
				id, ok = m["ID"]
			}
			if ok {
				group.DbID = id.(string)
			}
		}
	}

	if group.DbID != "" {
		//Update existing group status
		return s.db.UpdateRecord(groupmodel.DbStatusName, groupmodel.TableStatusName, group.DbID, status)
	}

	//Create new group entry
	grID, err := s.db.InsertRecord(groupmodel.DbStatusName, groupmodel.TableStatusName, status)
	if err != nil {
		group.DbID = grID
	}
	return err
}

func (s *GroupService) dumpGroupStatus(group Group) error {
	var leds []string
	for _, led := range group.Runtime.Leds {
		leds = append(leds, led.Mac)
	}

	var sensors []string
	for _, sensor := range group.Runtime.Sensors {
		sensors = append(sensors, sensor.Mac)
	}

	status := groupmodel.GroupStatus{
		Group:              group.Runtime.Group,
		Auto:               *group.Runtime.Auto,
		TimeToAuto:         group.TimeToAuto,
		SensorRule:         *group.Runtime.SensorRule,
		Error:              group.Error,
		Presence:           group.Presence,
		TimeToLeave:        group.PresenceTimeout,
		CorrectionInterval: *group.Runtime.CorrectionInterval,
		SetpointLeds:       group.Setpoint,
		SlopeStart:         *group.Runtime.SlopeStart,
		SlopeStop:          *group.Runtime.SlopeStop,
		Leds:               leds,
		Sensors:            sensors,
	}

	return s.updateDatabase(group, status)
}

func (s *GroupService) groupRun(group *Group) error {
	ticker := time.NewTicker(time.Second)
	go func() {
		counter := 0
		for {
			select {
			case events := <-group.Event:
				for eventType, e := range events {
					switch eventType {
					case EventStop:
						return

					case EventChange:
						group.updateConfig(e)

					case EventManual:
						rlog.Info("Received manual event ", group)
						group.TimeToAuto = *group.Runtime.Watchdog
						s.setpointLed(group)
						s.dumpGroupStatus(*group)
					}
				}
			case <-ticker.C:
				counter++
				// compute timetoAuto and switch back to Auto mode
				if group.TimeToAuto <= 0 {
					auto := true
					group.Runtime.Auto = &auto
				}
				if group.TimeToAuto > 0 {
					group.TimeToAuto--
				}
				if counter == *group.Runtime.CorrectionInterval {
					if *group.Runtime.Auto == true && group.Slope == 0 {
						s.computeSensorsValues(group)
						if group.Presence {
							if group.Runtime.GroupRules.Brightness != nil {
								readBrightness := *group.Runtime.GroupRules.Brightness
								if group.Brightness > readBrightness {
									group.NewSetpoint = group.Setpoint - group.Scale
									if group.NewSetpoint < 0 {
										group.NewSetpoint = 0
									}
									group.Slope = *group.Runtime.SlopeStop
								}
								if group.Brightness < readBrightness {
									group.NewSetpoint = group.Setpoint + group.Scale
									if group.NewSetpoint > 100 {
										group.NewSetpoint = 100
									}
									group.Slope = *group.Runtime.SlopeStart
								}
							}
						} else {
							//empty room
							group.NewSetpoint = 0
							group.Slope = *group.Runtime.SlopeStop
						}
					}
					counter = 0
				}
				s.setpointLed(group)
				err := s.dumpGroupStatus(*group)
				if err != nil {
					rlog.Errorf("Cannot dump status to database for " + strconv.Itoa(group.Runtime.Group) + " err " + err.Error())
				}
			}
		}
	}()
	return nil
}

func (s *GroupService) computeSensorsValues(group *Group) {
	//Fetch sensors value
	nbSensors := len(group.Runtime.Sensors)
	if nbSensors == 0 {
		return
	}
	var sensors []driversensor.Sensor
	for _, sensor := range group.Runtime.Sensors {
		criteria := make(map[string]interface{})
		criteria["Mac"] = sensor.Mac
		sensorStored, err := s.db.GetRecord(driversensor.DbName, driversensor.TableName, criteria)
		if err != nil || sensorStored == nil {
			continue
		}
		cell, _ := driversensor.ToSensor(sensorStored)
		if cell == nil {
			continue
		}
		sensors = append(sensors, *cell)
	}

	if len(sensors) == 0 {
		//No sensors fetch in database
		return
	}

	//compute sensor values
	presence := sensors[0].Presence
	group.Brightness = sensors[0].Brightness

	for i, sensor := range sensors {
		if i == 0 {
			continue
		}
		if sensor.Presence {
			presence = true
		}
		switch *group.Runtime.SensorRule {
		case groupmodel.SensorAverage:
			group.Brightness += sensor.Brightness / nbSensors
		case groupmodel.SensorMax:
			if group.Brightness < sensor.Brightness {
				group.Brightness = sensor.Brightness
			}
		case groupmodel.SensorMin:
			if group.Brightness > sensor.Brightness {
				group.Brightness = sensor.Brightness
			}
		}
	}

	// manage presence group timeout
	if group.Runtime.GroupRules.Presence != nil {
		if !presence {
			//TODO fix presence timeout: don't wait correctionInterval
			if group.PresenceTimeout <= 0 {
				group.PresenceTimeout = *group.Runtime.GroupRules.Presence
			} else {
				group.PresenceTimeout--
			}
			if group.PresenceTimeout == 0 {
				group.Presence = false
			}
		} else {
			group.Presence = true
		}
	} else {
		group.Presence = true
	}

}

func (s *GroupService) setpointLed(group *Group) {
	// Slope Computation
	diff := group.NewSetpoint - group.Setpoint
	if diff == 0 {
		return
	}
	if group.Slope > 0 {
		group.Setpoint += int(diff / group.Slope)
		group.Slope--
	} else {
		group.Setpoint = group.NewSetpoint
	}
	if group.Setpoint < 0 {
		group.Setpoint = 0
	}
	if group.Setpoint > 100 {
		group.Setpoint = 100
	}
	rlog.Info("Set brightness now to " + strconv.Itoa(group.Setpoint) + " , Remaining time " + strconv.Itoa(group.Slope))

	for _, led := range group.Runtime.Leds {
		if led.SwitchMac != s.mac {
			rlog.Info("Switch mac is " + s.mac + " and led is connected to switch " + led.SwitchMac + " skip it")
			continue
		}
		urlCmd := "/write/switch/led/update/settings"
		conf := driverled.LedConf{
			Mac:      led.Mac,
			Setpoint: &group.Setpoint,
		}
		dump, err := conf.ToJSON()
		if err != nil {
			rlog.Error("Cannot dump : ", err.Error())
			continue
		}
		s.broker.SendCommand(urlCmd, dump)
	}
}

func (s *GroupService) createGroup(runtime groupmodel.GroupRuntime) {
	group := Group{
		Event:   make(chan map[string]*groupmodel.GroupRuntime),
		Runtime: runtime,
		Scale:   10,
	}
	s.groups[runtime.Group] = group
	s.groupRun(&group)
}

func (s *GroupService) stopGroup(group groupmodel.GroupRuntime) {
	event := make(map[string]*groupmodel.GroupRuntime)
	event[EventStop] = nil
	s.groups[group.Group].Event <- event
}

func (s *GroupService) deleteGroup(group groupmodel.GroupRuntime) {
	s.stopGroup(group)
	time.Sleep(time.Second)

	gr, _ := s.groups[group.Group]
	if gr.DbID != "" {
		s.db.DeleteRecord(groupmodel.DbStatusName, groupmodel.TableStatusName, gr)
	}

	delete(s.groups, group.Group)
}

func (s *GroupService) reloadGroupConfig(groupID int, newconfig groupmodel.GroupRuntime) {
	event := make(map[string]*groupmodel.GroupRuntime)
	event[EventChange] = &newconfig
	s.groups[groupID].Event <- event
}

func (gr *Group) updateConfig(new *groupmodel.GroupRuntime) {
	if new == nil {
		return
	}
	if new.Auto != nil {
		gr.Runtime.Auto = new.Auto

		if *gr.Runtime.Auto == false && new.Setpoints != nil {
			if new.Setpoints.SpLeds != nil {
				gr.NewSetpoint = *new.Setpoints.SpLeds
				event := make(map[string]*groupmodel.GroupRuntime)
				event[EventManual] = nil
				gr.Event <- event
			}
		}
	}
	if new.CorrectionInterval != nil {
		gr.Runtime.CorrectionInterval = new.CorrectionInterval
	}
	if new.SensorRule != nil {
		gr.Runtime.SensorRule = new.SensorRule
	}
	if new.Leds != nil {
		gr.Runtime.Leds = new.Leds
	}
	if new.Sensors != nil {
		gr.Runtime.Sensors = new.Sensors
	}
	if new.SlopeStart != nil {
		gr.Runtime.SlopeStart = new.SlopeStart
	}
	if new.SlopeStop != nil {
		gr.Runtime.SlopeStop = new.SlopeStop
	}
	if new.GroupRules != nil {
		if gr.Runtime.GroupRules == nil {
			rules := groupmodel.Rule{}
			gr.Runtime.GroupRules = &rules
		}
		if new.GroupRules.Brightness != nil {
			gr.Runtime.GroupRules.Brightness = new.GroupRules.Brightness
		}
		if new.GroupRules.Presence != nil {
			gr.Runtime.GroupRules.Presence = new.GroupRules.Presence
		}
	}
}
