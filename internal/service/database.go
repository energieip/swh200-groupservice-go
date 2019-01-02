package service

import (
	"github.com/energieip/common-database-go/pkg/database"
	gr "github.com/energieip/common-group-go/pkg/groupmodel"
	pkg "github.com/energieip/common-service-go/pkg/service"
	"github.com/romana/rlog"
)

func (s *GroupService) prepareDatabase(conf pkg.ServiceConfig) error {
	db, err := database.NewDatabase(database.RETHINKDB)
	if err != nil {
		rlog.Error("database err " + err.Error())
		return err
	}

	confDb := database.DatabaseConfig{
		IP:   conf.DB.ClientIP,
		Port: conf.DB.ClientPort,
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

func (s *GroupService) updateDatabase(group Group, status gr.GroupStatus) error {

	if group.DbID == "" {
		//Fetch existing group status
		criteria := make(map[string]interface{})
		criteria["Group"] = group.Runtime.Group
		groupStored, err := s.db.GetRecord(gr.DbStatusName, gr.TableStatusName, criteria)
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
		return s.db.UpdateRecord(gr.DbStatusName, gr.TableStatusName, group.DbID, status)
	}

	//Create new group entry
	grID, err := s.db.InsertRecord(gr.DbStatusName, gr.TableStatusName, status)
	if err != nil {
		group.DbID = grID
	}
	return err
}
