package main

import (
	"context"
	"encoding/json"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"log"
	"net/http"
	"sync"
	"time"
)

// Plugin groups the methods a plugin needs
type Plugin struct {
	lock       sync.Mutex
}

type pluginSpec struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Interfaces  []string `json:"interfaces"`
	APIVersion  string   `json:"api_version,omitempty"`
}

type node struct {
	Latest      map[string]latest       `json:"latest"`
}

type metadataTemplate struct {
	ID       string		`json:"id"`
	Label    string		`json:"label,omitempty"`
	Format   string		`json:"format,omitempty"`
	Priority float64		`json:"priority,omitempty"`
}

type topology struct {
	Nodes           	map[string]node	`json:"nodes"`
	MetadataTemplate 	map[string]metadataTemplate 		`json:"metadata_templates"`
}

type report struct {
	Container 	topology
	Plugins 	[]pluginSpec
}

func (p *Plugin) metadataTemplates() map[string]metadataTemplate {
	return map[string]metadataTemplate{
		"affinity": {
			ID:       "affinity",
			Label:    "CPU Affinity",
			Format:   "latest",
			Priority: 0.1,
		},
	}
}

func getContainerCpuset(Id string) string{
	cli, err := client.NewClientWithOpts(client.WithHost("unix:///var/run/docker.sock"), client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	inspectResult, err := cli.ContainerInspect(context.Background(), Id)
	if err != nil {
		log.Fatalln("Failed to get container Cpuset")
	}

	if inspectResult.HostConfig.CpusetCpus == "" {
		return "All"
	}
	return inspectResult.HostConfig.CpusetCpus
}

func getContainerNodes() map[string]node {
	affinityMap := make(map[string]node)
	cli, err := client.NewClientWithOpts(client.WithHost("unix:///var/run/docker.sock"), client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All:true})
	if err != nil {
		panic(err)
	}

	for _, containerTmp := range containers{
		keyTmp := containerTmp.ID+";<container>"
		affinityMap[keyTmp] = node {
			map[string]latest {
				"affinity" :
				{
					Date:  time.Now(),
					Value: getContainerCpuset(containerTmp.ID),
				},
			},
		}
	}
	return affinityMap
}

type latest struct {
	Date	time.Time		`json:"date"`
	Value	string			`json:"value"`
}

func (p *Plugin) makeReport() (*report, error) {
	rpt := &report{
		Container: topology{
			Nodes: getContainerNodes(),
			MetadataTemplate: p.metadataTemplates(),
		},
		Plugins: []pluginSpec{
			{
				ID:          "epa",
				Label:       "epa",
				Description: "Show EPA information for container",
				Interfaces:  []string{"reporter"},
				APIVersion:  "1",
			},
		},
	}
	return rpt, nil
}

func (p *Plugin) Report(w http.ResponseWriter, r *http.Request) {
	p.lock.Lock()
	defer p.lock.Unlock()
	log.Println(r.URL.String())
	rpt, err := p.makeReport()
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	raw, err := json.Marshal(*rpt)
	if err != nil {
		log.Printf("error: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(raw); err != nil {
		log.Fatalln("Failed to write back report")
		return
	}
}