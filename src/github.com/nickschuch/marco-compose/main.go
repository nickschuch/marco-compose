package main

import (
	"errors"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/nickschuch/marco-lib"
	"github.com/samalba/dockerclient"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	name    = "compose"
	service = "io.docker.compose.service"
	project = "io.docker.compose.project"
)

var (
	cliMarco     = kingpin.Flag("marco", "The remote Marco backend.").Default("http://localhost:81").OverrideDefaultFromEnvar("MARCO_COMPOSE_URL").String()
	cliEndpoint  = kingpin.Flag("endpoint", "The Docker endpoint.").Default("unix:///var/run/docker.sock").OverrideDefaultFromEnvar("DOCKER_HOST").String()
	cliFrequency = kingpin.Flag("frequency", "How often to push to Marco").Default("15").OverrideDefaultFromEnvar("MARCO_COMPOSE_FREQUENCY").Int64()
	cliPorts     = kingpin.Flag("ports", "The ports you wish to proxy.").Default("80,8080,2368,8983").OverrideDefaultFromEnvar("MARCO_COMPOSE_PORTS").String()
	cliDomain    = kingpin.Flag("domain", "The base domain for all compose records..").Default("").OverrideDefaultFromEnvar("MARCO_COMPOSE_DOMAIN").String()
)

func main() {
	kingpin.Parse()

	for {
		Push(*cliMarco)
		time.Sleep(time.Duration(*cliFrequency) * time.Second)
	}
}

func Push(m string) {
	var b []marco.Backend

	log.WithFields(log.Fields{
		"type": "started",
	}).Info("Started pushing data to Marco.")

	// Get a list of backends keyed by the domain.
	list, err := getList()

	log.Info(list)

	if err != nil {
		log.WithFields(log.Fields{
			"type": "failed",
		}).Info(err)
		return
	}

	// Convert into the objects required for a push to Marco.
	for d, l := range list {
		n := marco.Backend{
			Type:   name,
			Domain: d,
			List:   l,
		}
		b = append(b, n)
	}

	// Attempt to send data to Marco.
	err = marco.Send(b, *cliMarco)
	if err != nil {
		log.WithFields(log.Fields{
			"type": "failed",
		}).Info(err)
		return
	}

	log.WithFields(log.Fields{
		"type": "completed",
	}).Info("Successfully pushed data to Marco.")
}

func getList() (map[string][]string, error) {
	// These are the URL's (keyed by domain) that we will return.
	list := make(map[string][]string)

	dockerClient, err := dockerclient.NewDockerClient(*cliEndpoint, nil)
	if err != nil {
		return list, err
	}

	containers, err := dockerClient.ListContainers(false, false, "")
	if err != nil {
		return list, err
	}

	for _, c := range containers {
		container, _ := dockerClient.InspectContainer(c.Id)

		// We try to find the domain based on the labels. If we don't have one
		// then we have nothing left to do with this container.
		envDomain, err := GetContainerDomain(c.Labels)
		if err != nil {
			continue
		}

		if *cliDomain != "" {
			envDomain = envDomain + "." + *cliDomain
		}

		// Here we build the proxy URL based on the exposed values provided
		// by NetworkSettings. If a container has not been exposed, it will
		// not work. We then build a URL based on these exposed values and:
		//   * Add a container reference so we can perform safe operations
		//     in the future.
		//   * Add the built url to the proxy lists for load balancing.
		for portString, portObject := range container.NetworkSettings.Ports {
			port := getPort(portString)
			if strings.Contains(*cliPorts, port) {
				url := getProxyUrl(portObject)
				if url != "" {
					list[envDomain] = append(list[envDomain], url)
				}
			}
		}
	}

	return list, nil
}

func GetContainerDomain(l map[string]string) (string, error) {
	if _, ok := l[service]; !ok {
		return "", errors.New("Cannot find service label")
	}

	if _, ok := l[project]; !ok {
		return "", errors.New("Cannot find project label")
	}

	return l[project] + "-" + l[service], nil
}

func getPort(exposed string) string {
	port := strings.Split(exposed, "/")
	return port[0]
}

func getProxyUrl(binding []dockerclient.PortBinding) string {
	// Ensure we have PortBinding values to build against.
	if len(binding) <= 0 {
		return ""
	}

	// Handle IP 0.0.0.0 the same way Swarm does. We replace this with an IP
	// that uses a local context.
	port := binding[0].HostPort
	ip := binding[0].HostIp
	if ip == "0.0.0.0" {
		ip = "127.0.0.1"
	}

	return "http://" + ip + ":" + port
}
