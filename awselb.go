package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elb"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
	mkr "github.com/mackerelio/mackerel-client-go"
)

// ELB type
type ELB struct {
	Name    string
	DNSName string
	HostID  string
}

func fetchLoadBalancerList(sess client.ConfigProvider) []*ELB {
	svc := elb.New(sess)

	params := &elb.DescribeLoadBalancersInput{}
	resp, err := svc.DescribeLoadBalancers(params)

	if err != nil {
		fmt.Println("fetchLoadBalancerList: ", err.Error())
		return []*ELB{}
	}

	//fmt.Println(resp)
	var elbs []*ELB
	for _, lbd := range resp.LoadBalancerDescriptions {
		elbs = append(elbs, &ELB{
			Name:    *lbd.LoadBalancerName,
			DNSName: *lbd.DNSName,
		})
		//return elbs
	}
	return elbs
}

var graphdefs = map[string](mp.Graphs){
	"elb.hostcount": mp.Graphs{
		Label: "Host Count",
		Unit:  "integer",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "HealthyHostCount", Label: "Healthy Host Count", Diff: false},
			mp.Metrics{Name: "UnHealthyHostCount", Label: "UnHealthy Host Count", Diff: false},
		},
	},
	"elb.httpcode": mp.Graphs{
		Label: "HTTP Code Count",
		Unit:  "integer",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "HTTPCode_Backend_2XX", Label: "Backend 2XX", Diff: false},
			mp.Metrics{Name: "HTTPCode_Backend_3XX", Label: "Backend 3XX", Diff: false},
			mp.Metrics{Name: "HTTPCode_Backend_4XX", Label: "Backend 4XX", Diff: false},
			mp.Metrics{Name: "HTTPCode_Backend_5XX", Label: "Backend 5XX", Diff: false},
			mp.Metrics{Name: "HTTPCode_ELB_4XX", Label: "ELB 4XX", Diff: false},
			mp.Metrics{Name: "HTTPCode_ELB_5XX", Label: "ELB 5XX", Diff: false},
		},
	},
	"elb.latency": mp.Graphs{
		Label: "Latency",
		Unit:  "float",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "Latency", Label: "Latency", Diff: false},
		},
	},
	"elb.requestcount": mp.Graphs{
		Label: "Request Count",
		Unit:  "integer",
		Metrics: [](mp.Metrics){
			mp.Metrics{Name: "RequestCount", Label: "Request Count", Diff: false},
		},
	},
}

func getELBsMetricStatistics(sess client.ConfigProvider, elbs []*ELB) {
	for _, elb := range elbs {
		getELBMetricStatistics(sess, elb)
	}
}

func getELBMetricStatistics(sess client.ConfigProvider, elb *ELB) []*mkr.MetricValue {
	svc := cloudwatch.New(sess)
	/*
		metricnames := []string{
			"HealthyHostCount", "UnHealthyHostCount",
			"HTTPCode_Backend_2XX", "HTTPCode_Backend_3XX", "HTTPCode_Backend_4XX", "HTTPCode_Backend_5XX",
			"HTTPCode_ELB_4XX", "HTTPCode_ELB_5XX",
			"Latency",
			"RequestCount",
			"BackendConnectionErrors",
			"SurgeQueueLength",
			"SpilloverCount",
		}
	*/

	var metricValues []*mkr.MetricValue
	for key, graphdef := range graphdefs {
		for _, metrics := range graphdef.Metrics {

			params := &cloudwatch.GetMetricStatisticsInput{
				EndTime:    aws.Time(time.Now()),
				MetricName: aws.String(metrics.Name),
				Namespace:  aws.String("AWS/ELB"),
				Period:     aws.Int64(60),
				StartTime:  aws.Time(time.Now().Add(-1 * time.Minute)),
				Statistics: []*string{
					aws.String("Sum"),
				},
				Dimensions: []*cloudwatch.Dimension{
					{
						Name:  aws.String("LoadBalancerName"),
						Value: aws.String(elb.Name),
					}},
			}
			resp, err := svc.GetMetricStatistics(params)

			if err != nil {
				fmt.Println(err.Error())
				return metricValues
			}

			for _, dp := range resp.Datapoints {
				fmt.Println(elb.Name, metrics.Name, *(dp.Sum))
				metricValues = append(metricValues, &mkr.MetricValue{
					Name:  "custom." + key + "." + metrics.Name,
					Value: *(dp.Sum),
					Time:  dp.Timestamp.Unix(),
				})
			}
		}
	}
	return metricValues
}
