package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
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
	SNSTopic string `json:"snstopic"`
}

// HandleRequest Handle lambda function
func HandleRequest(ctx context.Context) (string, error) {
	var buffer bytes.Buffer
	/* Read config */
	// Open our jsonFile
	jsonFile, err := os.Open("config.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}
	// fmt.Println("Successfully Opened config.json")
	buffer.WriteString("Successfully Opened config.json\n")
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

	// fmt.Println("---------Jenkins node status...---------")
	buffer.WriteString("---------Jenkins node status...---------\n")

	for _, node := range nodes {

		// Fetch Node Data
		node.Poll()
		status, err := node.IsOnline()
		if err != nil {
			panic(err)
		}
		if status {
			// fmt.Println(fmt.Sprintf("Node %s is Online. No of executors %s, Is idle? %t", node.GetName(), strconv.FormatInt(node.Raw.NumExecutors, 10), node.Raw.Idle))
			buffer.WriteString(fmt.Sprintf("Node %s is Online. No of executors %s, Is idle? %t\n", node.GetName(), strconv.FormatInt(node.Raw.NumExecutors, 10), node.Raw.Idle))
		} else {
			// fmt.Println("Node " + node.GetName() + " is  Offline")
			buffer.WriteString("Node " + node.GetName() + " is  Offline\n")
		}
	}

	// fmt.Println("---------Jenkins build queue status...---------")
	buffer.WriteString("---------Jenkins build queue status...---------\n")

	jqueue, err := jenkins.GetQueue()
	if err != nil {
		panic(err)
	}

	if len(jqueue.Tasks()) == 0 {
		// fmt.Println("No jobs queued")
		buffer.WriteString("No jobs queued\n")

	} else {
		for _, task := range jqueue.Raw.Items {
			now := time.Now()
			secs := now.Unix()
			// fmt.Println(fmt.Sprintf("Task ID: %d, Task Name: %s, Why: %s, In Queue Since %d minutes", task.ID, task.Task.Name, task.Why, secs/60-task.InQueueSince/60000))
			buffer.WriteString(fmt.Sprintf("Task ID: %d, Task Name: %s, Why: %s, In Queue Since %d minutes\n", task.ID, task.Task.Name, task.Why, secs/60-task.InQueueSince/60000))
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

	// Send SNS notification
	//Create a session object to talk to SNS (also make sure you have your key and secret setup in your .aws/credentials file)
	svc := sns.New(session.New())
	// params will be sent to the publish call included here is the bare minimum params to send a message.
	params := &sns.PublishInput{
		Message:  aws.String(buffer.String()),                     // This is the message itself (can be XML / JSON / Text - anything you want)
		TopicArn: aws.String(connections.Connections[0].SNSTopic), //Get this from the Topic in the AWS console.
	}

	resp, err := svc.Publish(params) //Call to puclish the message

	if err != nil { //Check for errors
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return "Failed", nil
	}

	// Pretty-print the response data.
	fmt.Println(resp)

	return "Ok", nil
}

func main() {
	os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
	HandleRequest(nil)
	// lambda.Start(HandleRequest)
}
