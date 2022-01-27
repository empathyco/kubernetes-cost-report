// Package pkg provides functions to calculate co2.
package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// WattCPU returns the watt of a machine.
type WattCPU struct {
	Idle    float64
	Watt10  float64
	Watt50  float64
	Watt100 float64
}

// WattMemory returns the watt of a machine.
type WattMemory struct {
	Idle    float64
	Watt10  float64
	Watt50  float64
	Watt100 float64
}

// Co2Machine Co2Machine struct.
type Co2Machine struct {
	MachineType string
	TotalCPU    int
	TotalMemory int
	WattCPU     WattCPU
	WattMemory  WattMemory
	Co2Factory  float64
}

// Co2Metrics co2 metrics representation.
type Co2Metrics interface {
	PrintData()
}

// Co2PodMetrics represents the co2 consumption of a pod.
type Co2PodMetrics struct {
	Metrics PodMetrics
	Watts   float64 `json:"watts"`
	Co2     float64 `json:"co2"`
}

// PrintData print co2 data.
func (co2PodMetrics *Co2PodMetrics) PrintData() string {
	return fmt.Sprintf("%s\t\t\t\t\t%f\t%f\n",
		co2PodMetrics.Metrics.Name,
		co2PodMetrics.Watts,
		co2PodMetrics.Co2)
}

// Co2NodeMetrics represents the co2 consumption of a node.
type Co2NodeMetrics struct {
	Metrics NodeMetrics `json:"metrics"`
	WattCPU float64     `json:"co2_cpu"`
	WattMem float64     `json:"co2_memory"`
	Region  string      `json:"region"`
	Watts   float64     `json:"watts"`
	Co2     float64     `json:"co2"`
}

// PrintData Prints the data in the struct.
func (co2Metrics *Co2NodeMetrics) PrintData() string {
	return fmt.Sprintf("%s\t%s\t%f\t%f\t%f\t%f\n",
		co2Metrics.Metrics.Name,
		co2Metrics.Metrics.MachineType,
		co2Metrics.Metrics.CPUUsage,
		co2Metrics.Metrics.MemoryUsage,
		co2Metrics.Watts,
		co2Metrics.Co2)
}

// GetWattCPU Retrieves the co2 consumption of a machine.
func (c *Co2Machine) GetWattCPU(usage float64) float64 {
	if usage <= 10 {
		return c.WattCPU.Idle
	}
	if usage < 50 {
		return c.WattCPU.Idle + c.WattCPU.Watt10
	}
	if usage < 100 {
		return c.WattCPU.Idle + c.WattCPU.Watt50
	}

	return c.WattCPU.Idle + c.WattCPU.Watt100
}

// GetWattMem get watt per memeory unit.
func (c *Co2Machine) GetWattMem(usage float64) float64 {
	if usage <= 10 {
		return c.WattMemory.Idle
	}
	if usage < 50 {
		return c.WattMemory.Idle + c.WattMemory.Watt10
	}
	if usage < 100 {
		return c.WattMemory.Idle + c.WattMemory.Watt50
	}

	return c.WattMemory.Idle + c.WattMemory.Watt100
}

// GetWattCPUUnit Retrieves the co2 consumption of a vCPU in a node.
func (c *Co2Machine) GetWattCPUUnit(usage float64) float64 {
	return c.GetWattCPU(usage) / float64(c.TotalCPU)
}

// GetWattMemUnit Retrieves the co2 consumption of a Gb in a node.
func (c *Co2Machine) GetWattMemUnit(usage float64) float64 {
	return c.GetWattMem(usage) / float64(c.TotalMemory)
}

// GetWattComsumption Retrieves the watt consumption of a machine.
func (c *Co2Machine) GetWattComsumption(usageCPU, usageMem float64) float64 {
	return c.GetWattCPU(usageCPU) + c.GetWattMem(usageMem)
}

// Co2Region represetn a region aws.
type Co2Region struct {
	Region string
	Co2    float64
	PUE    float64
}

// PrintData Prints the data in the struct.
func (co2Metrics *Co2Region) PrintData() string {
	return fmt.Sprintf("%s\t%f\t%f\n",
		co2Metrics.Region,
		co2Metrics.PUE,
		co2Metrics.Co2)
}

// Co2DB Co2DB struct.
type Co2DB struct {
	MachineDB map[string]Co2Machine
	RegionsDB map[string]Co2Region
	Context   context.Context
}

// GetClient Retrieve a token, saves the token, then returns the generated client.
func GetClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	tokFile := "token.json"
	tok, err := TokenFromFile(tokFile)
	if err != nil {
		tok = GetTokenFromWeb(config)
		err = SaveToken(tokFile, tok)
		if err != nil {
			log.Fatalf("Unable to save token to file: %v", err)
		}
	}

	return config.Client(context.Background(), tok)
}

// GetTokenFromWeb Request a token from the web, then returns the retrieved token.
func GetTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	log.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	return tok
}

// TokenFromFile Retrieves a token from a local file.
func TokenFromFile(file string) (*oauth2.Token, error) {
	repoFile := filepath.Clean(file)
	fil, err := os.Open(repoFile)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	tok := &oauth2.Token{}
	err = json.NewDecoder(fil).Decode(tok)
	if err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	err = fil.Close()
	if err != nil {
		return nil, fmt.Errorf("close credentials: %w", err)
	}
	return tok, nil
}

// SaveToken a token to a file path.
func SaveToken(path string, token *oauth2.Token) error {
	log.Printf("Saving credential file to: %s\n", path)
	repoFile := filepath.Clean(path)
	// nolint:gofumpt
	fil, err := os.OpenFile(repoFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}

	err = json.NewEncoder(fil).Encode(token)
	if err != nil {
		return fmt.Errorf("decode credentials: %w", err)
	}

	err = fil.Close()
	if err != nil {
		return fmt.Errorf("close credentials: %w", err)
	}

	return nil
}

// NewCo2DB Create a new Co2DB.
func NewCo2DB(ctx context.Context) *Co2DB {
	co2DB := &Co2DB{
		MachineDB: make(map[string]Co2Machine),
		RegionsDB: make(map[string]Co2Region),
		Context:   ctx,
	}
	if err := co2DB.UpdateData(); err != nil {
		log.Fatalf("Unable to update data: %v", err)
	}

	return co2DB
}

// UpdateData Update the data from the API sheet.
func (c *Co2DB) UpdateData() error {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		return fmt.Errorf("get credentials: %w", err)
	}
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		return fmt.Errorf("get sheet config: %w", err)
	}
	client := GetClient(config)

	srv, err := sheets.NewService(c.Context, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("get sheet service: %w", err)
	}

	spreadsheetID := "1DqYgQnEDLQVQm5acMAhLgHLD8xXCG9BIrk-_Nv6jF3k"
	waitingGroup := sync.WaitGroup{}
	waitingGroup.Add(2)
	go func() {
		defer waitingGroup.Done()
		c.GetMachines(srv, spreadsheetID)
	}()
	go func() {
		defer waitingGroup.Done()
		c.GetRegions(srv, spreadsheetID)
	}()
	waitingGroup.Wait()

	return nil
}

// GetRegions Retrieves the regions from the API sheet.
func (c *Co2DB) GetRegions(srv *sheets.Service, spreadsheetID string) {
	readRange := "AWS Regions Mix Intensity!A2:G"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	for _, row := range resp.Values {
		if region := row[0].(string); region != "" {
			co2 := calcWatt(row[4].(string))
			pue := calcWatt(row[6].(string))
			c.RegionsDB[region] = Co2Region{
				Region: region,
				Co2:    co2,
				PUE:    pue,
			}
		}
	}
}

// GetMachines Get the machines from the API sheet.
func (c *Co2DB) GetMachines(srv *sheets.Service, spreadsheetID string) {
	readRange := "EC2 Instances Dataset!A2:AK"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	for _, row := range resp.Values {
		machineType := row[0].(string)
		cpu, _ := strconv.Atoi(row[2].(string))
		memory, _ := strconv.Atoi(row[5].(string))
		CPUWattIdle := calcWatt(row[14].(string))
		CPUWatt10 := calcWatt(row[15].(string))
		CPUWatt50 := calcWatt(row[16].(string))
		CPUWatt100 := calcWatt(row[17].(string))
		memoryWattIdle := calcWatt(row[18].(string))
		memoryWatt10 := calcWatt(row[19].(string))
		memoryWatt50 := calcWatt(row[20].(string))
		memoryWatt100 := calcWatt(row[21].(string))
		co2Factory := calcWatt(row[36].(string))
		c.MachineDB[machineType] = Co2Machine{
			MachineType: machineType,
			TotalCPU:    cpu,
			TotalMemory: memory,
			WattCPU:     WattCPU{Idle: CPUWattIdle, Watt10: CPUWatt10, Watt50: CPUWatt50, Watt100: CPUWatt100},
			WattMemory:  WattMemory{Idle: memoryWattIdle, Watt10: memoryWatt10, Watt50: memoryWatt50, Watt100: memoryWatt100},
			Co2Factory:  co2Factory,
		}
	}
}

// GetNodeWattComp returns the watt of a machine.
func (c *Co2DB) GetNodeWattComp(no NodeMetrics) float64 {
	watt := c.MachineDB[no.MachineType]

	return watt.GetWattComsumption(no.CPUUsage, no.MemoryUsage)
}

// GetPodsConsumption returns the co2 consumption of a pod.
func (c *Co2DB) GetPodsConsumption(metrics *Metrics) []Co2PodMetrics {
	co2NodesMetrics := c.GetNodesConsumption(metrics)
	co2PodsMetrics := make([]Co2PodMetrics, 0)
	for podIndex := range metrics.Pods {
		nodeMetrics := co2NodesMetrics[metrics.Pods[podIndex].Node]
		regionName := nodeMetrics.Region

		region := c.RegionsDB[regionName]
		watt := CalcPodWatt(&metrics.Pods[podIndex], &nodeMetrics)

		co2 := CalculateCo2(watt, 0.0, region)
		podCo2 := Co2PodMetrics{
			Metrics: metrics.Pods[podIndex],
			Watts:   watt,
			Co2:     co2,
		}
		co2PodsMetrics = append(co2PodsMetrics, podCo2)
	}

	return co2PodsMetrics
}

// CalcPodWatt calculates the watt consumption of a pod.
func CalcPodWatt(pod *PodMetrics, node *Co2NodeMetrics) float64 {
	expo := float64(1000000000)

	return (pod.CPUUsage/1000)*node.WattCPU + (pod.MemoryUsage/expo)*node.WattMem
}

// GetNodesConsumption returns the co2 consumption of each node.
func (c *Co2DB) GetNodesConsumption(m *Metrics) map[string]Co2NodeMetrics {
	nodesMetrics := make(map[string]Co2NodeMetrics)
	for _, node := range m.Nodes {
		watt := c.GetNodeWattComp(node)
		region := c.RegionsDB[node.Region]
		machine := c.MachineDB[node.MachineType]
		co2 := CalculateCo2(watt, c.MachineDB[node.MachineType].Co2Factory, region)

		co2Metrics := Co2NodeMetrics{
			Metrics: node,
			Watts:   watt,
			Co2:     co2,
			Region:  node.Region,
			WattCPU: machine.GetWattCPUUnit(node.CPUUsage),
			WattMem: machine.GetWattMemUnit(node.MemoryUsage),
		}
		nodesMetrics[node.Name] = co2Metrics
	}

	return nodesMetrics
}

// CalculateCo2 calculates the co2 emission of a given watt consumption.
func CalculateCo2(watt, co2MachineFactory float64, region Co2Region) float64 {
	return (watt/1000)*region.PUE*region.Co2 + co2MachineFactory
}

func calcWatt(cpuCo2 string) float64 {
	// strings.Replace(CPUCo2, ",", ".", -1)
	CPUCo2Float, err := strconv.ParseFloat(strings.ReplaceAll(cpuCo2, ",", "."), 64)
	if err != nil {
		log.Fatalf("Unable to convert string to int: %v", err)
	}

	return CPUCo2Float
}

// PrintWatt print watts.
func (c *Co2DB) PrintWatt() {
	for i, v := range c.MachineDB {
		log.Printf("%s\t%v\n", i, v)
	}
}

// GetMetrics get prometheus metrics.
func (c *Co2DB) GetMetrics(metrics *Metrics) prometheus.Gatherer {
	reg := prometheus.NewRegistry()
	labelNodes := []string{"name", "region", "machine_type"}
	labelPods := []string{"name", "region"}

	co2Nodes := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "co2_node",
		Help: "Cost Instance Type",
	}, labelNodes)

	wattNodes := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "watt_node",
		Help: "Watt Instance Type",
	}, labelNodes)

	co2Pods := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "co2_pod",
		Help: "Cost Pod",
	}, labelPods)

	wattPods := promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
		Name: "watt_pod",
		Help: "Watt Pods",
	}, labelPods)

	c.GetNodesMetrics(metrics, co2Nodes, wattNodes)
	c.GetPodsMetrics(metrics, co2Pods, wattPods)

	return reg
}

// GetNodesMetrics get metrics for nodes.
func (c *Co2DB) GetNodesMetrics(m *Metrics, co2Prom, wattProm *prometheus.GaugeVec) {
	for _, no := range m.Nodes {
		watt := c.GetNodeWattComp(no)
		co2 := CalculateCo2(watt, c.MachineDB[no.MachineType].Co2Factory, c.RegionsDB[no.Region])
		co2Prom.WithLabelValues(no.Name, no.Region, no.MachineType).Set(co2)
		wattProm.WithLabelValues(no.Name, no.Region, no.MachineType).Set(watt)
	}
}

// GetPodsMetrics get pod metrics.
func (c *Co2DB) GetPodsMetrics(metrics *Metrics, co2Prom, wattProm *prometheus.GaugeVec) {
	co2NodesMetrics := c.GetNodesConsumption(metrics)
	for pod := range metrics.Pods {
		nodeMetrics := co2NodesMetrics[metrics.Pods[pod].Node]
		regionName := nodeMetrics.Region
		region := c.RegionsDB[regionName]
		watt := CalcPodWatt(&metrics.Pods[pod], &nodeMetrics)

		co2 := CalculateCo2(watt, 0.0, region)
		co2Prom.WithLabelValues(metrics.Pods[pod].Name, regionName).Set(co2)
		wattProm.WithLabelValues(metrics.Pods[pod].Name, regionName).Set(watt)
	}
}
