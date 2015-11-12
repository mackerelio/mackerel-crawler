package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elb"

	mkr "github.com/mackerelio/mackerel-client-go"
	"github.com/mackerelio/mkr/logger"
)

// ELB type
type ELB struct {
	Name    string
	DNSName string
	HostID  string
}

// AWSSession ...
type AWSSession struct {
	Sess client.ConfigProvider
}

// NewAWSSession ...
func NewAWSSession(sess client.ConfigProvider) *AWSSession {
	return &AWSSession{Sess: sess}
}

func (s *AWSSession) fetchLoadBalancerList() []*ELB {
	svc := elb.New(s.Sess)

	params := &elb.DescribeLoadBalancersInput{}
	resp, err := svc.DescribeLoadBalancers(params)

	if err != nil {
		fmt.Println("fetchLoadBalancerList: ", err.Error())
		return []*ELB{}
	}

	var elbs []*ELB
	for _, lbd := range resp.LoadBalancerDescriptions {
		elbs = append(elbs, &ELB{
			Name:    *lbd.LoadBalancerName,
			DNSName: *lbd.DNSName,
		})
	}
	return elbs
}

var graphdefs = map[string](Graphs){
	"elb.hostcount": Graphs{
		Label: "Host Count",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "HealthyHostCount", Label: "Healthy Host Count"},
			Metrics{Name: "UnHealthyHostCount", Label: "UnHealthy Host Count"},
		},
	},
	"elb.httpcode": Graphs{
		Label: "HTTP Code Count",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "HTTPCode_Backend_2XX", Label: "Backend 2XX", Statistics: "Sum"},
			Metrics{Name: "HTTPCode_Backend_3XX", Label: "Backend 3XX", Statistics: "Sum"},
			Metrics{Name: "HTTPCode_Backend_4XX", Label: "Backend 4XX", Statistics: "Sum"},
			Metrics{Name: "HTTPCode_Backend_5XX", Label: "Backend 5XX", Statistics: "Sum"},
			Metrics{Name: "HTTPCode_ELB_4XX", Label: "ELB 4XX", Statistics: "Sum"},
			Metrics{Name: "HTTPCode_ELB_5XX", Label: "ELB 5XX", Statistics: "Sum"},
		},
	},
	"elb.latency": Graphs{
		Label: "Latency",
		Unit:  "float",
		Metrics: [](Metrics){
			Metrics{Name: "Latency", Label: "Latency"},
		},
	},
	"elb.requestcount": Graphs{
		Label: "Request Count",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "RequestCount", Label: "Request Count", Statistics: "Sum"},
		},
	},
}

func (s *AWSSession) getELBsMetricStatistics(elbs []*ELB) {
	for _, elb := range elbs {
		s.getELBMetricStatistics(elb)
	}
}

func (s *AWSSession) getELBMetricStatistics(elb *ELB) []*mkr.MetricValue {
	svc := cloudwatch.New(s.Sess)
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

			statistics := metrics.Statistics
			if metrics.Statistics == "" {
				statistics = "Average"
			}
			params := &cloudwatch.GetMetricStatisticsInput{
				EndTime:    aws.Time(time.Now()),
				MetricName: aws.String(metrics.Name),
				Namespace:  aws.String("AWS/ELB"),
				Period:     aws.Int64(60),
				StartTime:  aws.Time(time.Now().Add(-1 * time.Minute)),
				Statistics: []*string{
					aws.String(statistics),
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
				var value float64
				switch statistics {
				case "Sum":
					value = *(dp.Sum)
				default:
					value = *(dp.Average)
				}
				fmt.Println(elb.Name, metrics.Name, value)
				metricValues = append(metricValues, &mkr.MetricValue{
					Name:  "custom." + key + "." + metrics.Name,
					Value: value,
					Time:  dp.Timestamp.Unix(),
				})
			}
		}
	}
	return metricValues
}

func (s *AWSSession) updateELBList(client *mkr.Client) []*ELB {
	elbs := s.fetchLoadBalancerList()
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

func (s *AWSSession) crawlELBMetrics(client *mkr.Client, elbs []*ELB) {
	for _, elb := range elbs {
		metricValues := s.getELBMetricStatistics(elb)
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
