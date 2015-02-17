package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
)

type ExperimentManagerConnector struct {
	login                    string
	password                 string
	experimentManagerAddress string
	client                   *http.Client
	scheme                   string
}

func NewExperimentManagerConnector(login, password, certificatePath, scheme string, insecure bool) *ExperimentManagerConnector {
	var client *http.Client
	tlsConfig := tls.Config{InsecureSkipVerify: insecure}

	if certificatePath != "" {
		CA_Pool := x509.NewCertPool()
		severCert, err := ioutil.ReadFile(certificatePath)
		if err != nil {
			log.Fatal("An error occured: could not load Scalarm certificate")
		}
		CA_Pool.AppendCertsFromPEM(severCert)

		tlsConfig.RootCAs = CA_Pool
	}

	client = &http.Client{Transport: &http.Transport{TLSClientConfig: &tlsConfig}}

	return &ExperimentManagerConnector{login: login, password: password, client: client, scheme: scheme}
}

func (emc *ExperimentManagerConnector) GetExperimentManagerLocation(informationServiceAddress string) error {
	resp, err := emc.client.Get(emc.scheme + "://" + informationServiceAddress + "/experiment_managers")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Printf(string(body))
	var experimentManagerAddresses []string
	err = json.Unmarshal(body, &experimentManagerAddresses)
	if err != nil {
		return err
	}

	emc.experimentManagerAddress = experimentManagerAddresses[0] // TODO random
	log.Printf("\texp_man_address: " + emc.experimentManagerAddress)
	return nil
}

type EMJsonResponse struct {
	Status     string
	Sm_records []Sm_record
}

func (emc *ExperimentManagerConnector) GetSimulationManagerRecords(infrastructure string) ([]Sm_record, error) {
	urlString := emc.scheme + "://" + emc.experimentManagerAddress + "/simulation_managers?"
	params := url.Values{}
	params.Add("infrastructure", infrastructure)
	params.Add("options", "{\"states_not\":\"error\",\"onsite_monitoring\":true}")
	urlString = urlString + params.Encode()

	request, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(emc.login, emc.password)

	resp, err := emc.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Printf("URL: %s\nBODY: %s\n", urlString, body)
	var response EMJsonResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	if response.Status != "ok" {
		return nil, errors.New("Damaged data")
	}

	return response.Sm_records, nil
}

func (emc *ExperimentManagerConnector) GetSimulationManagerCode(smRecordId string, infrastructure string) error {
	debug.FreeOSMemory()
	urlString := emc.scheme + "://" + emc.experimentManagerAddress + "/simulation_managers/" + smRecordId + "/code?"
	params := url.Values{}
	params.Add("infrastructure", infrastructure)
	urlString = urlString + params.Encode()

	request, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return err
	}
	request.SetBasicAuth(emc.login, emc.password)

	resp, err := emc.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("sources_"+smRecordId+".zip", body, 0600)
	if err != nil {
		return err
	}

	return nil
}

func inner_sm_record_marshal(current, old, name string, comma *bool, parameters *bytes.Buffer) {
	if current != old {
		if *comma {
			parameters.WriteString(",")
		}
		parameters.WriteString("\"" + name + "\":\"" + escape(current) + "\"")
		*comma = true
	}
}

func sm_record_marshal(sm_record, old_sm_record *Sm_record) string {
	var parameters bytes.Buffer
	parameters.WriteString("{")
	comma := false

	inner_sm_record_marshal(sm_record.State, old_sm_record.State, "state", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Resource_status, old_sm_record.Resource_status, "resource_status", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Cmd_to_execute, old_sm_record.Cmd_to_execute, "cmd_to_execute", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Cmd_to_execute_code, old_sm_record.Cmd_to_execute_code, "cmd_to_execute_code", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Error_log, old_sm_record.Error_log, "error_log", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Job_id, old_sm_record.Job_id, "job_id", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Pid, old_sm_record.Pid, "pid", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Vm_id, old_sm_record.Vm_id, "vm_id", &comma, &parameters)

	inner_sm_record_marshal(sm_record.Res_id, old_sm_record.Res_id, "res_id", &comma, &parameters)

	parameters.WriteString("}")

	log.Printf("Update: " + parameters.String())
	return parameters.String()
}

func (emc *ExperimentManagerConnector) NotifyStateChange(sm_record, old_sm_record *Sm_record, infrastructure string) error { //do zmiany

	// sm_json, err := json.Marshal(sm_record)
	// if err != nil {
	// 	return err
	// }
	// log.Printf(string(sm_json))
	// data := url.Values{"parameters": {string(sm_json)}, "infrastructure": {infrastructure}}

	//----
	data := url.Values{"parameters": {sm_record_marshal(sm_record, old_sm_record)}, "infrastructure": {infrastructure}}
	//----

	urlString := emc.scheme + "://" + emc.experimentManagerAddress + "/simulation_managers/" + sm_record.Id

	request, err := http.NewRequest("PUT", urlString, strings.NewReader(data.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	request.SetBasicAuth(emc.login, emc.password)

	resp, err := emc.client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return nil
	} else {
		log.Printf("Status code: " + strconv.Itoa(resp.StatusCode))
		return errors.New("Update failed")
	}
	return nil
}

func escape(input string) string {
	output := strings.Replace(input, "\n", "\\n", -1)
	output = strings.Replace(output, "\r", "\\r", -1)
	output = strings.Replace(output, "\t", "\\t", -1)
	output = strings.Replace(output, `'`, `\'`, -1)
	output = strings.Replace(output, `"`, `\"`, -1)

	return output
}
