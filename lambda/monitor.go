package main

import (
	"fmt"
	"github.com/bndr/gojenkins"
)

func main() {
	jenkins, _ := gojenkins.CreateJenkins(nil, "http://url/", "user", "pass").Init()

	nodes, err := jenkins.GetAllNodes()
	if err != nil {
		panic(err)
	}

	for _, node := range nodes {

		// Fetch Node Data
		node.Poll()
		status, err := node.IsOnline()
		if err != nil {
			panic(err)
		}
		if status {
			fmt.Println("Node " + node.GetName() + " is Online")
		} else {
			fmt.Println("Node " + node.GetName() + " is  Offline")
		}
	}

	jqueue, err := jenkins.GetQueue()
	if err != nil {
		panic(err)
	}

	for _, task := range jqueue.Tasks() {
		fmt.Println(task.GetWhy())
	}

	jobs, err := jenkins.GetAllJobs()
	if err != nil {
		panic(err)
	}

	for _, job := range jobs {
		fmt.Println(fmt.Sprintf("Job Name: %s - Buildable: %t", job.GetName(), job.GetDetails().Buildable))
	}

}
