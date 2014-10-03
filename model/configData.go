package model

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"scalarm_monitoring_daemon/utils"
)

type ConfigData struct {
	InformationServiceAddress string
	Login                     string
	Password                  string
	Infrastructures           []string
	ScalarmCertificatePath    string
	ScalarmScheme             string
}

func ReadConfiguration() (*ConfigData, error) {
	log.Printf("readConfiguration")

	data, err := ioutil.ReadFile("config.json")
	utils.Check(err)

	var configData ConfigData
	err = json.Unmarshal(data, &configData)
	utils.Check(err)

	if configData.ScalarmCertificatePath[0] == '~' {
		configData.ScalarmCertificatePath = os.Getenv("HOME") + configData.ScalarmCertificatePath[1:]
	}

	if configData.ScalarmScheme == "" {
		configData.ScalarmScheme = "https"
	}

	log.Printf("\tInformation Service address: %v", configData.InformationServiceAddress)
	log.Printf("\tlogin:                       %v", configData.Login)
	log.Printf("\tpassword:                    %v", configData.Password)
	log.Printf("\tinfrastructures:             %v", configData.Infrastructures)
	log.Printf("\tScalarm certificate path:    %v", configData.ScalarmCertificatePath)
	log.Printf("\tScalarm scheme:              %v", configData.ScalarmScheme)

	log.Printf("readConfiguration: OK")
	return &configData, nil
}
