package main

import (
	"os"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/codegangsta/cli"
	mkr "github.com/mackerelio/mackerel-client-go"
)

// Metrics ...
type Metrics struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Statistics string `json:"statistics"`
}

// Graphs ...
type Graphs struct {
	Label   string    `json:"label"`
	Unit    string    `json:"unit"`
	Metrics []Metrics `json:"metrics"`
}

func doAction(c *cli.Context) {
	awsKeyID := c.String("aws-key-id")
	awsSecKey := c.String("aws-secret-key")
	mackerelAPIKey := c.String("mackerel-api-key")

	client := mkr.NewClient(mackerelAPIKey)

	sess := session.New(&aws.Config{
		Credentials: credentials.NewStaticCredentials(awsKeyID, awsSecKey, ""),
		Region:      aws.String("ap-northeast-1"),
	})
	awsSession := NewAWSSession(sess)

	if c.Bool("debug") {
		sess.Config.LogLevel = aws.LogLevel(aws.LogDebug)
	}

	elbs := awsSession.updateELBList(client)

	tickChan := time.NewTicker(60 * time.Second)
	quit := make(chan struct{})

	for {
		select {
		case <-tickChan.C:
			awsSession.crawlELBMetrics(client, elbs)
		case <-quit:
			tickChan.Stop()
			return
		}
	}

	//listMetric(sess)
}

func main() {
	app := cli.NewApp()
	app.Name = "mackerel-fetcher"
	// app.Version = Version
	app.Usage = "Metric fetcher for mackerel.io"
	app.Author = "Hatena Co., Ltd."
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "aws-key-id",
			Usage:  "AWS Key ID",
			EnvVar: "AWS_ACCESS_KEY_ID",
		},
		cli.StringFlag{
			Name:   "aws-secret-key",
			Usage:  "AWS Secret Key",
			EnvVar: "AWS_SECRET_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "mackerel-api-key",
			Usage:  "Mackerel API Key",
			EnvVar: "MACKEREL_APIKEY",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "Debug",
		},
	}
	app.Action = doAction

	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu)

	app.Run(os.Args)
}
