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
	Metrics      map[string]metric       `json:"metric"`
}

type metadataTemplate struct {
	ID       string  `json:"id"`
	Label    string  `json:"label,omitempty"`
	Format   string  `json:"format,omitempty"`
	Priority float64 `json:"priority,omitempty"`
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

func getContainerMounts() map[string]string {
	affinityMap := make(map[string]string)
	cli, err := client.NewClientWithOpts(client.WithHost("unix:///var/run/docker.sock"), client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All:true})
	if err != nil {
		panic(err)
	}

	for _, container := range containers {
		keyTmp := container.ID+"<container>"
		affinityMap[keyTmp] = string(len(container.Mounts))
	}
	return affinityMap
}

func (p *Plugin) getTopologyHost() string {
	return "abcdefgh;<container>"
}

type metric struct {
	Date  time.Time `json:"date"`
	Value string   `json:"value"`
}

func (p *Plugin) metrics() metric {
	value := "0-7"
	return metric{
		Date : time.Now(),
		Value: value,
	}
}


func (p *Plugin) makeReport() (*report, error) {
	metrics := p.metrics()
	rpt := &report{
		Container: topology{
			Nodes: map[string]node{
				p.getTopologyHost(): {
					map[string]metric {
						"affinity" :
							metrics,
					},
				},
			},
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