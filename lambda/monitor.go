package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/bndr/gojenkins"
)

// Connections struct which contains
// an array of users
type Connections struct {
	Connections []Auth `json:"connections"`
}

// Auth struct which contains a name
// a type and a list of social links
type Auth struct {
	URL      string `json:"url"`
	UserName string `json:"username"`
	Password string `json:"password"`
}

func main() {

	/* Read config */
	// Open our jsonFile
	jsonFile, err := os.Open("config.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Successfully Opened config.json")
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened xmlFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	var connections Connections

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	json.Unmarshal(byteValue, &connections)

	jenkins, _ := gojenkins.CreateJenkins(nil, connections.Connections[0].URL,
		connections.Connections[0].UserName, connections.Connections[0].Password).Init()

	nodes, err := jenkins.GetAllNodes()
	if err != nil {
		panic(err)
	}

	fmt.Println("---------Jenkins node status...---------")

	for _, node := range nodes {

		// Fetch Node Data
		node.Poll()
		status, err := node.IsOnline()
		if err != nil {
			panic(err)
		}
		if status {
			fmt.Println(fmt.Sprintf("Node %s is Online. No of executors %s, Is idle? %t", node.GetName(), strconv.FormatInt(node.Raw.NumExecutors, 10), node.Raw.Idle))
		} else {
			fmt.Println("Node " + node.GetName() + " is  Offline")
		}
	}

	fmt.Println("---------Jenkins build queue status...---------")

	jqueue, err := jenkins.GetQueue()
	if err != nil {
		panic(err)
	}

	if len(jqueue.Tasks()) == 0 {
		fmt.Println("No jobs queued")

	} else {
		for _, task := range jqueue.Tasks() {
			now := time.Now()
			secs := now.Unix()
			fmt.Println("Task :" + task.Raw.Task.Name + " " + task.GetWhy() + " " + "In Queue since: " + strconv.FormatInt(secs/60-task.Raw.InQueueSince/60000, 10) + " minutes")
		}
	}

	//fmt.Println("---------Jenkins jobs...---------")
	//
	//jobs, err := jenkins.GetAllJobs()
	//if err != nil {
	//	panic(err)
	//}
	//
	//for _, job := range jobs {
	//	fmt.Println(fmt.Sprintf("Job Name: %s - Buildable: %t", job.GetName(), job.GetDetails().Buildable))
	//}

}
