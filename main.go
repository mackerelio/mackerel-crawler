package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/codegangsta/cli"
	mkr "github.com/mackerelio/mackerel-client-go"
	"github.com/mackerelio/mkr/logger"
)

func updateELBList(sess client.ConfigProvider, client *mkr.Client) []*ELB {
	elbs := fetchLoadBalancerList(sess)
	for _, elb := range elbs {
		hosts, err := client.FindHosts(&mkr.FindHostsParam{Name: elb.DNSName})
		if err != nil {
			logger.Log("error", fmt.Sprintf("Mackerel FindHosts: %s", err.Error()))
			continue
		}

		if len(hosts) == 1 {
			elb.HostID = hosts[0].ID
			logger.Log("info", fmt.Sprintf("Host Found: %s -> %s", hosts[0].ID, hosts[0].Name))
		}
		if len(hosts) == 0 {
			elb.HostID, err = client.CreateHost(&mkr.CreateHostParam{
				Name: elb.DNSName,
			})
			if err != nil {
				logger.Log("error", fmt.Sprintf("Mackerel CreateHost: %s", err.Error()))
			}
		}
	}
	return elbs
}

func crawlELBMetrics(sess client.ConfigProvider, client *mkr.Client, elbs []*ELB) {
	for _, elb := range elbs {
		metricValues := getELBMetricStatistics(sess, elb)
		logger.Log("info", fmt.Sprintf("%s", metricValues))
		err := client.PostHostMetricValuesByHostID(elb.HostID, metricValues)
		//logger.DieIf(err)
		if err != nil {
			logger.Log("error", err.Error())
		}

		for _, metric := range metricValues {
			logger.Log("thrown", fmt.Sprintf("%s '%s\t%f\t%d'", elb.HostID, metric.Name, metric.Value, metric.Time))
		}
	}
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

	if c.Bool("debug") {
		sess.Config.LogLevel = aws.LogLevel(aws.LogDebug)
	}

	elbs := updateELBList(sess, client)

	tickChan := time.NewTicker(60 * time.Second)
	quit := make(chan struct{})

	for {
		select {
		case <-tickChan.C:
			crawlELBMetrics(sess, client, elbs)
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
