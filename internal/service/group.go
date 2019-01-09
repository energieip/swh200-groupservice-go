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
	Event           chan map[string]*groupmodel.GroupConfig
	Runtime         groupmodel.GroupConfig
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

func (s *GroupService) dumpGroupStatus(group Group) error {
	name := ""
	if group.Runtime.FriendlyName != nil {
		name = *group.Runtime.FriendlyName
	}
	correctionInterval := 1
	if group.Runtime.CorrectionInterval != nil {
		correctionInterval = *group.Runtime.CorrectionInterval
	}
	auto := false
	if group.Runtime.Auto != nil {
		auto = *group.Runtime.Auto
	}
	slopeStart := 0
	slopeStop := 0
	if group.Runtime.SlopeStart != nil {
		slopeStart = *group.Runtime.SlopeStart
	}
	if group.Runtime.SlopeStop != nil {
		slopeStop = *group.Runtime.SlopeStop
	}
	sensorRule := groupmodel.SensorAverage
	if group.Runtime.SensorRule != nil {
		sensorRule = *group.Runtime.SensorRule
	}
	watchdog := 0
	if group.Runtime.Watchdog != nil {
		watchdog = *group.Runtime.Watchdog
	}
	status := groupmodel.GroupStatus{
		Group:              group.Runtime.Group,
		Auto:               auto,
		TimeToAuto:         group.TimeToAuto,
		SensorRule:         sensorRule,
		Error:              group.Error,
		Presence:           group.Presence,
		TimeToLeave:        group.PresenceTimeout,
		CorrectionInterval: correctionInterval,
		SetpointLeds:       group.Setpoint,
		SlopeStart:         slopeStart,
		SlopeStop:          slopeStop,
		Leds:               group.Runtime.Leds,
		Sensors:            group.Runtime.Sensors,
		RuleBrightness:     group.Runtime.RuleBrightness,
		RulePresence:       group.Runtime.RulePresence,
		Watchdog:           watchdog,
		FriendlyName:       name,
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
				if group.TimeToAuto <= 0 && (group.Runtime.Auto == nil || *group.Runtime.Auto == false) {
					auto := true
					group.Runtime.Auto = &auto
					rlog.Info("Switch group " + strconv.Itoa(group.Runtime.Group) + " back to Automatic mode")
				}
				if group.TimeToAuto > 0 {
					group.TimeToAuto--
				}

				// Do not wait for correction interval to re-adjust the sensor values
				s.computeSensorsValues(group)

				if group.Runtime.CorrectionInterval == nil || counter == *group.Runtime.CorrectionInterval {
					if (group.Runtime.Auto != nil && *group.Runtime.Auto == true) && group.Slope == 0 {
						if group.Presence {
							if group.Runtime.RuleBrightness != nil {
								readBrightness := *group.Runtime.RuleBrightness
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
							rlog.Info("Room is now empty for group", strconv.Itoa(group.Runtime.Group))
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
		criteria["Mac"] = sensor
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
	if group.Runtime.RulePresence != nil {
		if !presence {
			if group.PresenceTimeout <= 0 {
				group.PresenceTimeout = *group.Runtime.RulePresence
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
		urlCmd := "/write/switch/led/update/settings"
		conf := driverled.LedConf{
			Mac:          led,
			SetpointAuto: &group.Setpoint,
		}
		dump, err := conf.ToJSON()
		if err != nil {
			rlog.Error("Cannot dump : ", err.Error())
			continue
		}
		s.broker.SendCommand(urlCmd, dump)
	}
}

func (s *GroupService) createGroup(runtime groupmodel.GroupConfig) {
	if runtime.Auto == nil {
		auto := true
		runtime.Auto = &auto
	}
	group := Group{
		Event:   make(chan map[string]*groupmodel.GroupConfig),
		Runtime: runtime,
		Scale:   10,
	}
	s.groups[runtime.Group] = group
	s.groupRun(&group)
}

func (s *GroupService) stopGroup(group groupmodel.GroupConfig) {
	event := make(map[string]*groupmodel.GroupConfig)
	event[EventStop] = nil
	s.groups[group.Group].Event <- event
}

func (s *GroupService) deleteGroup(group groupmodel.GroupConfig) {
	s.stopGroup(group)
	time.Sleep(time.Second)

	gr, _ := s.groups[group.Group]
	if gr.DbID != "" {
		s.db.DeleteRecord(groupmodel.DbStatusName, groupmodel.TableStatusName, gr)
	}

	delete(s.groups, group.Group)
}

func (s *GroupService) reloadGroupConfig(groupID int, newconfig groupmodel.GroupConfig) {
	event := make(map[string]*groupmodel.GroupConfig)
	event[EventChange] = &newconfig
	s.groups[groupID].Event <- event
}

func (gr *Group) updateConfig(new *groupmodel.GroupConfig) {
	if new == nil {
		return
	}
	if new.Auto != gr.Runtime.Auto {
		gr.Runtime.Auto = new.Auto
	}

	if gr.Runtime.Auto != nil && *gr.Runtime.Auto == false && new.SetpointLeds != nil {
		go func() {
			gr.NewSetpoint = *new.SetpointLeds
			event := make(map[string]*groupmodel.GroupConfig)
			event[EventManual] = nil
			gr.Event <- event
		}()
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
	if new.RuleBrightness != nil {
		gr.Runtime.RuleBrightness = new.RuleBrightness
	}
	if new.RulePresence != nil {
		gr.Runtime.RulePresence = new.RulePresence
	}
	if new.FriendlyName != nil {
		gr.Runtime.FriendlyName = new.FriendlyName
	}
	if new.SensorRule != nil {
		gr.Runtime.SensorRule = new.SensorRule
	}
	if new.Watchdog != nil {
		gr.Runtime.Watchdog = new.Watchdog
	}
}
