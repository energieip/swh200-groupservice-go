package service

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/energieip/common-database-go/pkg/database"
	gr "github.com/energieip/common-group-go/pkg/groupmodel"
	"github.com/energieip/common-network-go/pkg/network"
	"github.com/energieip/swh200-groupservice-go/pkg/config"
	"github.com/energieip/swh200-groupservice-go/pkg/tools"
	"github.com/romana/rlog"
)

//GroupService content
type GroupService struct {
	db     database.DatabaseInterface
	broker network.NetworkInterface //Local Broker for internal service communication
	mac    string                   //Switch mac address
	groups map[int]Group
}

func (s *GroupService) onSetup(client network.Client, msg network.Message) {
	var groups map[int]gr.GroupRuntime
	err := json.Unmarshal(msg.Payload(), &groups)
	if err != nil {
		rlog.Error("Error during parsing", err.Error())
		return
	}
	for grID, group := range groups {
		if _, ok := s.groups[grID]; ok {
			rlog.Warn("Group " + strconv.Itoa(grID) + " already exists skip it")
			continue
		}
		s.createGroup(group)
	}
}

func (s *GroupService) onUpdate(client network.Client, msg network.Message) {
	var groups map[int]gr.GroupRuntime
	err := json.Unmarshal(msg.Payload(), &groups)
	if err != nil {
		rlog.Error("Error during parsing", err.Error())
		return
	}
	for grID, group := range groups {
		if _, ok := s.groups[grID]; !ok {
			rlog.Warn("Group " + strconv.Itoa(grID) + " not found skip it")
			continue
		}
		s.reloadGroupConfig(grID, group)
	}
}

func (s *GroupService) onRemove(client network.Client, msg network.Message) {
	var groups map[int]gr.GroupRuntime
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

func (s *GroupService) prepareDatabase(conf *config.Configuration) error {
	db, err := database.NewDatabase(database.RETHINKDB)
	if err != nil {
		rlog.Error("database err " + err.Error())
		return err
	}

	confDb := database.DatabaseConfig{
		IP:   conf.DatabaseIP,
		Port: conf.DatabasePort,
	}
	err = db.Initialize(confDb)
	if err != nil {
		rlog.Error("Cannot connect to database " + err.Error())
		return err
	}
	s.db = db

	err = s.db.CreateDB(gr.DbStatusName)
	if err != nil {
		rlog.Warn("Create DB " + gr.DbStatusName + " err: " + err.Error())
	}

	err = s.db.CreateTable(gr.DbStatusName, gr.TableStatusName, &gr.GroupStatus{})
	if err != nil {
		rlog.Warn("Create table ", err.Error())
	}
	return nil
}

func (s *GroupService) prepareNetwork(conf *config.Configuration) error {
	hostname, err := os.Hostname()
	if err != nil {
		rlog.Error("Cannot read hostname " + err.Error())
		return err
	}
	clientID := "Group" + hostname
	driversBroker, err := network.NewNetwork(network.MQTT)
	if err != nil {
		rlog.Error("Cannot connect to broker " + conf.DriversBrokerIP + " error: " + err.Error())
		return err
	}
	s.broker = driversBroker

	callbacks := make(map[string]func(client network.Client, msg network.Message))
	callbacks["/write/switch/group/setup/config"] = s.onSetup
	callbacks["/write/switch/group/update/settings"] = s.onUpdate
	callbacks["/remove/switch/group/update/settings"] = s.onRemove

	confDrivers := network.NetworkConfig{
		IP:         conf.DriversBrokerIP,
		Port:       conf.DriversBrokerPort,
		ClientName: clientID,
		Callbacks:  callbacks,
		LogLevel:   *conf.LogLevel,
	}
	err = driversBroker.Initialize(confDrivers)
	if err != nil {
		rlog.Error("Cannot connect to broker " + conf.DriversBrokerIP + " error: " + err.Error())
	} else {
		rlog.Info(clientID + " connected to drivers broker " + conf.DriversBrokerIP)
	}
	return err
}

//Initialize service
func (s *GroupService) Initialize(confFile string) error {
	s.groups = make(map[int]Group, 0)
	s.mac = strings.ToUpper(strings.Replace(tools.GetMac(), ":", "", -1))

	conf, err := config.ReadConfig(confFile)
	if err != nil {
		rlog.Error("Cannot parse configuration file " + err.Error())
		return err
	}
	os.Setenv("RLOG_LOG_LEVEL", *conf.LogLevel)
	os.Setenv("RLOG_LOG_NOTIME", "yes")
	rlog.UpdateEnv()
	rlog.Info("Starting Group service")

	err = s.prepareDatabase(conf)
	if err != nil {
		return err
	}

	s.prepareNetwork(conf)
	if err != nil {
		return err
	}

	rlog.Info("Group service started")
	return nil
}

//Stop service
func (s *GroupService) Stop() {
	rlog.Info("Stopping Group service")
	for _, group := range s.groups {
		s.stopGroup(group.Runtime)
	}
	s.broker.Disconnect()
	s.db.Close()
	rlog.Info("Group service stopped")
}

//Run service mainloop
func (s *GroupService) Run() error {
	select {}
}
