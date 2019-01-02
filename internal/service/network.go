package service

import (
	"encoding/json"
	"os"
	"strconv"

	gm "github.com/energieip/common-group-go/pkg/groupmodel"
	"github.com/energieip/common-network-go/pkg/network"
	pkg "github.com/energieip/common-service-go/pkg/service"
	"github.com/romana/rlog"
)

func (s *GroupService) onUpdate(client network.Client, msg network.Message) {
	var groups map[int]gm.GroupConfig
	err := json.Unmarshal(msg.Payload(), &groups)
	if err != nil {
		rlog.Error("Error during parsing", err.Error())
		return
	}
	for grID, group := range groups {
		if _, ok := s.groups[grID]; !ok {
			rlog.Info("Group " + strconv.Itoa(grID) + " create it")
			s.createGroup(group)
			continue
		}
		rlog.Info("Group " + strconv.Itoa(grID) + " reload it")
		s.reloadGroupConfig(grID, group)
	}
}

func (s *GroupService) onRemove(client network.Client, msg network.Message) {
	var groups map[int]gm.GroupConfig
	err := json.Unmarshal(msg.Payload(), &groups)
	if err != nil {
		rlog.Error("Error during parsing", err.Error())
		return
	}
	for grID := range groups {
		if group, ok := s.groups[grID]; ok {
			s.deleteGroup(group.Runtime)
		}
	}
}

func (s *GroupService) prepareNetwork(conf pkg.ServiceConfig) error {
	hostname, err := os.Hostname()
	if err != nil {
		rlog.Error("Cannot read hostname " + err.Error())
		return err
	}
	clientID := "Group" + hostname
	driversBroker, err := network.NewNetwork(network.MQTT)
	if err != nil {
		rlog.Error("Cannot connect to broker " + conf.LocalBroker.IP + " error: " + err.Error())
		return err
	}
	s.broker = driversBroker

	callbacks := make(map[string]func(client network.Client, msg network.Message))
	callbacks["/write/switch/group/update/settings"] = s.onUpdate
	callbacks["/remove/switch/group/update/settings"] = s.onRemove

	confDrivers := network.NetworkConfig{
		IP:         conf.LocalBroker.IP,
		Port:       conf.LocalBroker.Port,
		ClientName: clientID,
		Callbacks:  callbacks,
		LogLevel:   conf.LogLevel,
	}
	err = driversBroker.Initialize(confDrivers)
	if err != nil {
		rlog.Error("Cannot connect to broker " + conf.LocalBroker.IP + " error: " + err.Error())
	} else {
		rlog.Info(clientID + " connected to drivers broker " + conf.LocalBroker.IP)
	}
	return err
}
