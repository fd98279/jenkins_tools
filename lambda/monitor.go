package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
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
	URL                 string `json:"url"`
	UserName            string `json:"username"`
	Password            string `json:"password"`
	SNSTopic            string `json:"snstopic"`
	BuildQueueTimeLimit int64  `json:"buildqueuetimelimit"`
}

// HandleMessage Print to console and update the SNS message buffer
func HandleMessage(debug *bool, buffer *bytes.Buffer, msg string) {
	if *debug {
		fmt.Println(msg)
	}
	buffer.WriteString(msg)
}

// HandleRequest Handle lambda function
func HandleRequest(ctx context.Context) (string, error) {

	configFile := flag.String("config", "./config.json", "config file path")
	debug := flag.Bool("debug", true, "debug mode")
	flag.Parse()

	/* Read config */
	jsonFile, err := os.Open(*configFile)

	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	/* Create buffer to be used for SNS topic notification */
	var buffer bytes.Buffer

	HandleMessage(debug, &buffer, "Successfully Opened config.json\n")
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

	/* 	fmt.Println("Starting the application...")
	   	response, err := http.Get(connections.Connections[0].URL)
	   	if err != nil {
	   		fmt.Printf("The HTTP request failed with error %s\n", err)
	   	} else {
	   		data, _ := ioutil.ReadAll(response.Body)
	   		fmt.Println(string(data))
	   	}
	*/

	nodes, err := jenkins.GetAllNodes()
	if err != nil {
		panic(err)
	}

	HandleMessage(debug, &buffer, "---------Jenkins node status...---------\n")

	for _, node := range nodes {

		// Fetch Node Data
		node.Poll()
		status, err := node.IsOnline()
		if err != nil {
			panic(err)
		}
		if status {
			HandleMessage(debug, &buffer, fmt.Sprintf("Node %s is Online. No of executors %s, Is idle? %t", node.GetName(), strconv.FormatInt(node.Raw.NumExecutors, 10), node.Raw.Idle))
		} else {
			HandleMessage(debug, &buffer, "Node "+node.GetName()+" is  Offline\n")
		}
	}

	HandleMessage(debug, &buffer, "---------Jenkins build queue status...---------")

	jqueue, err := jenkins.GetQueue()
	if err != nil {
		panic(err)
	}

	jobStuck := false

	if len(jqueue.Tasks()) == 0 {
		HandleMessage(debug, &buffer, "No jobs queued\n")
	} else {
		for _, task := range jqueue.Raw.Items {
			now := time.Now()
			secs := now.Unix()
			HandleMessage(debug, &buffer, fmt.Sprintf("Task ID: %d, Task Name: %s, Why: %s, In Queue Since %d minutes", task.ID, task.Task.Name, task.Why, secs/60-task.InQueueSince/60000))
			if secs/60-task.InQueueSince/60000 > connections.Connections[0].BuildQueueTimeLimit {
				jobStuck = true
			}
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

	// Send SNS notification if any job is stuck
	// Create a session object to talk to SNS (also make sure you have your key and secret setup in your .aws/credentials file)
	if jobStuck {
		HandleMessage(debug, &buffer, fmt.Sprintf("Jobs stuck\n"))
		os.Setenv("AWS_SDK_LOAD_CONFIG", "true")
		HandleMessage(debug, &buffer, "Sending SNS notification\n")
		svc := sns.New(session.New())
		// params will be sent to the publish call included here is the bare minimum params to send a message.
		params := &sns.PublishInput{
			Message:  aws.String(buffer.String()),                     // This is the message itself (can be XML / JSON / Text - anything you want)
			TopicArn: aws.String(connections.Connections[0].SNSTopic), //Get this from the Topic in the AWS console.
		}

		resp, err := svc.Publish(params) //Call to publish the message

		if err != nil { //Check for errors
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			HandleMessage(debug, &buffer, fmt.Sprintf("SNS notification send error: %s\n", err.Error()))
			return "Failed", nil
		}
		//  Pretty-print the response data.
		HandleMessage(debug, &buffer, fmt.Sprintf("SNS notification: Response: %s\n", resp))

	} else {
		HandleMessage(debug, &buffer, "No jobs stuck\n")
	}

	if ctx != nil {
		ctx.Done()
	}

	return "Ok", nil
}

func main() {
	// HandleRequest(nil)
	lambda.Start(HandleRequest)
}
