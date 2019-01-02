package service

import (
	"os"
	"strings"

	"github.com/energieip/common-database-go/pkg/database"
	"github.com/energieip/common-network-go/pkg/network"
	pkg "github.com/energieip/common-service-go/pkg/service"
	"github.com/energieip/common-tools-go/pkg/tools"
	"github.com/romana/rlog"
)

//GroupService content
type GroupService struct {
	db     database.DatabaseInterface
	broker network.NetworkInterface //Local Broker for internal service communication
	mac    string                   //Switch mac address
	groups map[int]Group
}

//Initialize service
func (s *GroupService) Initialize(confFile string) error {
	s.groups = make(map[int]Group, 0)
	s.mac = strings.ToUpper(strings.Replace(tools.GetMac(), ":", "", -1))

	conf, err := pkg.ReadServiceConfig(confFile)
	if err != nil {
		rlog.Error("Cannot parse configuration file " + err.Error())
		return err
	}
	os.Setenv("RLOG_LOG_LEVEL", conf.LogLevel)
	os.Setenv("RLOG_LOG_NOTIME", "yes")
	rlog.UpdateEnv()
	rlog.Info("Starting Group service")

	err = s.prepareDatabase(*conf)
	if err != nil {
		return err
	}

	s.prepareNetwork(*conf)
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
