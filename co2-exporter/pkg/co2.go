package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

type WattCpu struct {
	Idle    float64
	Watt10  float64
	Watt50  float64
	Watt100 float64
}

type WattMemory struct {
	Idle    float64
	Watt10  float64
	Watt50  float64
	Watt100 float64
}

type Co2Machine struct {
	MachineType string
	TotalCpu    int
	TotalMemory int
	WattCpu     WattCpu
	WattMemory  WattMemory
	Co2Factory  float64
}

type Co2Metrics interface {
	PrintData()
}

type Co2PodMetrics struct {
	Metrics PodMetrics
	Watts   float64 `json:"watts"`
	Co2     float64 `json:"co2"`
}

func (co2PodMetrics *Co2PodMetrics) PrintData() string {
	return fmt.Sprintf("%s\t\t\t\t\t%f\t%f\n",
		co2PodMetrics.Metrics.Name,
		co2PodMetrics.Watts,
		co2PodMetrics.Co2)

}

type Co2NodeMetrics struct {
	Metrics NodeMetrics `json:"metrics"`
	WattCPU float64     `json:"co2_cpu"`
	WattMem float64     `json:"co2_memory"`
	Region  string      `json:"region"`
	Watts   float64     `json:"watts"`
	Co2     float64     `json:"co2"`
}

func (co2Metrics *Co2NodeMetrics) PrintData() string {
	return fmt.Sprintf("%s\t%s\t%f\t%f\t%f\t%f\n",
		co2Metrics.Metrics.Name,
		co2Metrics.Metrics.MachineType,
		co2Metrics.Metrics.CPUUsage,
		co2Metrics.Metrics.MemoryUsage,
		co2Metrics.Watts,
		co2Metrics.Co2)
}

func (c *Co2Machine) GetWattCpu(usage float64) float64 {
	if usage <= 10 {
		return c.WattCpu.Idle
	}
	if usage < 50 {
		return c.WattCpu.Idle + c.WattCpu.Watt10
	}
	if usage < 100 {
		return c.WattCpu.Idle + c.WattCpu.Watt50
	}
	return c.WattCpu.Idle + c.WattCpu.Watt100
}

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

func (c *Co2Machine) GetWattCPUUnit(usage float64) float64 {
	return c.GetWattCpu(usage) / float64(c.TotalCpu)
}

func (c *Co2Machine) GetWattMemUnit(usage float64) float64 {
	return c.GetWattMem(usage) / float64(c.TotalMemory)
}

func (c *Co2Machine) GetWattComsumption(usageCpu, usageMem float64) float64 {
	return c.GetWattCpu(usageCpu) + c.GetWattMem(usageMem)
}

type Co2Region struct {
	Region string
	Co2    float64
	PUE    float64
}

func (co2Metrics *Co2Region) PrintData() string {
	return fmt.Sprintf("%s\t%f\t%f\n",
		co2Metrics.Region,
		co2Metrics.PUE,
		co2Metrics.Co2)
}

type Co2DB struct {
	MachineDB map[string]Co2Machine
	RegionsDb map[string]Co2Region
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
		SaveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// GetTokenFromWeb Request a token from the web, then returns the retrieved token.
func GetTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
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
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func SaveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func NewCo2DB(ctx context.Context) *Co2DB {
	co2DB := &Co2DB{
		MachineDB: make(map[string]Co2Machine),
		RegionsDb: make(map[string]Co2Region),
		Context:   ctx,
	}
	co2DB.UpdateData()
	return co2DB
}

func (c *Co2DB) UpdateData() error {
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		return err
	}
	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets.readonly")
	if err != nil {
		return err
	}
	client := GetClient(config)

	srv, err := sheets.NewService(c.Context, option.WithHTTPClient(client))
	if err != nil {
		return err
	}

	spreadsheetId := "1DqYgQnEDLQVQm5acMAhLgHLD8xXCG9BIrk-_Nv6jF3k"

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		c.GetMachines(srv, spreadsheetId)
	}()
	go func() {
		defer wg.Done()
		c.GetRegions(srv, spreadsheetId)
	}()
	wg.Wait()
	return nil
}

func (c *Co2DB) GetRegions(srv *sheets.Service, spreadsheetId string) {
	readRange := "AWS Regions Mix Intensity!A2:G"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from sheet: %v", err)
	}

	for _, row := range resp.Values {
		if region := row[0].(string); region != "" {
			co2 := calcWatt(row[4].(string))
			pue := calcWatt(row[6].(string))
			c.RegionsDb[region] = Co2Region{
				Region: region,
				Co2:    co2,
				PUE:    pue,
			}
		}

	}
}

func (c *Co2DB) GetMachines(srv *sheets.Service, spreadsheetId string) {
	readRange := "EC2 Instances Dataset!A2:AK"
	resp, err := srv.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
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
			TotalCpu:    cpu,
			TotalMemory: memory,
			WattCpu:     WattCpu{Idle: CPUWattIdle, Watt10: CPUWatt10, Watt50: CPUWatt50, Watt100: CPUWatt100},
			WattMemory:  WattMemory{Idle: memoryWattIdle, Watt10: memoryWatt10, Watt50: memoryWatt50, Watt100: memoryWatt100},
			Co2Factory:  co2Factory,
		}
	}
}

func (c *Co2DB) GetNodeWattComp(no NodeMetrics) float64 {
	watt := c.MachineDB[no.MachineType]
	return watt.GetWattComsumption(no.CPUUsage, no.MemoryUsage)
}

func (c *Co2DB) GetPodsConsumption(m *Metrics) []Co2PodMetrics {
	co2NodesMetrics := c.GetNodesConsumption(m)
	co2PodsMetrics := make([]Co2PodMetrics, 0)
	for _, pod := range m.Pods {
		nodeMetrics := co2NodesMetrics[pod.Node]
		regionName := nodeMetrics.Region

		region := c.RegionsDb[regionName]
		watt := CalcPodWatt(&pod, &nodeMetrics)

		co2 := CalculateCo2(watt, 0.0, region)
		podCo2 := Co2PodMetrics{
			Metrics: pod,
			Watts:   watt,
			Co2:     co2,
		}
		co2PodsMetrics = append(co2PodsMetrics, podCo2)

	}
	return co2PodsMetrics
}

func CalcPodWatt(pod *PodMetrics, node *Co2NodeMetrics) float64 {
	expo := float64(1000000000)
	return (pod.CPUUsage/1000)*node.WattCPU + (pod.MemoryUsage/expo)*node.WattMem
}

func (c *Co2DB) GetNodesConsumption(m *Metrics) map[string]Co2NodeMetrics {
	nodesMetrics := make(map[string]Co2NodeMetrics)
	for _, no := range m.Nodes {
		watt := c.GetNodeWattComp(no)
		region := c.RegionsDb[no.Region]
		machine := c.MachineDB[no.MachineType]
		co2 := CalculateCo2(watt, c.MachineDB[no.MachineType].Co2Factory, region)

		co2Metrics := Co2NodeMetrics{
			Metrics: no,
			Watts:   watt,
			Co2:     co2,
			Region:  no.Region,
			WattCPU: machine.GetWattCPUUnit(no.CPUUsage),
			WattMem: machine.GetWattMemUnit(no.MemoryUsage),
		}
		nodesMetrics[no.Name] = co2Metrics
	}
	return nodesMetrics
}

func CalculateCo2(watt, co2MachineFactory float64, region Co2Region) float64 {
	return (watt/1000)*region.PUE*region.Co2 + co2MachineFactory
}

func calcWatt(CPUCo2 string) float64 {
	// strings.Replace(CPUCo2, ",", ".", -1)
	CPUCo2Float, err := strconv.ParseFloat(strings.Replace(CPUCo2, ",", ".", -1), 64)
	if err != nil {
		log.Fatalf("Unable to convert string to int: %v", err)
	}
	return float64(CPUCo2Float)
}

func (c *Co2DB) PrintWatt() {
	for i, v := range c.MachineDB {
		fmt.Printf("%s\t%v\n", i, v)
	}
}

func (c *Co2DB) GetMetrics(m *Metrics) prometheus.Gatherer {
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

	c.GetNodesMetrics(m, co2Nodes, wattNodes)
	c.GetPodsMetrics(m, co2Pods, wattPods)

	return reg
}

func (c *Co2DB) GetNodesMetrics(m *Metrics, co2Prom, wattProm *prometheus.GaugeVec) {
	for _, no := range m.Nodes {
		watt := c.GetNodeWattComp(no)
		co2 := CalculateCo2(watt, c.MachineDB[no.MachineType].Co2Factory, c.RegionsDb[no.Region])
		co2Prom.WithLabelValues(no.Name, no.Region, no.MachineType).Set(co2)
		wattProm.WithLabelValues(no.Name, no.Region, no.MachineType).Set(watt)
	}
}

func (c *Co2DB) GetPodsMetrics(m *Metrics, co2Prom, wattProm *prometheus.GaugeVec) {
	co2NodesMetrics := c.GetNodesConsumption(m)
	for _, pod := range m.Pods {
		nodeMetrics := co2NodesMetrics[pod.Node]
		regionName := nodeMetrics.Region
		region := c.RegionsDb[regionName]
		watt := CalcPodWatt(&pod, &nodeMetrics)

		co2 := CalculateCo2(watt, 0.0, region)
		co2Prom.WithLabelValues(pod.Name, regionName).Set(co2)
		wattProm.WithLabelValues(pod.Name, regionName).Set(watt)
	}
}
