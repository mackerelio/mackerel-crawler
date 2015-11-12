package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/rds"

	mkr "github.com/mackerelio/mackerel-client-go"
	"github.com/mackerelio/mkr/logger"
)

// AWSElement type
type AWSElement struct {
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

func (s *AWSSession) fetchLoadBalancerList() []*AWSElement {
	svc := elb.New(s.Sess)

	params := &elb.DescribeLoadBalancersInput{}
	resp, err := svc.DescribeLoadBalancers(params)

	if err != nil {
		fmt.Println("fetchLoadBalancerList: ", err.Error())
		return []*AWSElement{}
	}

	var elbs []*AWSElement
	for _, lbd := range resp.LoadBalancerDescriptions {
		elbs = append(elbs, &AWSElement{
			Name:    *lbd.LoadBalancerName,
			DNSName: *lbd.DNSName,
		})
	}
	return elbs
}

func (s *AWSSession) fetchRDSList() []*AWSElement {
	svc := rds.New(s.Sess)

	params := &rds.DescribeDBInstancesInput{}
	resp, err := svc.DescribeDBInstances(params)

	if err != nil {
		fmt.Println("fetchLoadBalancerList: ", err.Error())
		return []*AWSElement{}
	}

	//fmt.Println(resp)
	var rdss []*AWSElement
	for _, rds := range resp.DBInstances {
		rdss = append(rdss, &AWSElement{
			Name:    *rds.DBInstanceIdentifier,
			DNSName: *rds.Endpoint.Address,
		})
	}
	return rdss
}

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

// http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/elb-metricscollected.html
var elbGraphdefs = map[string](Graphs){
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

/*
CPUUtilization rds.cpu.user.percentage
CPUCreditUsage rds.cpucredit.usage
CPUCreditBalance rds.cpucredit.balance

FreeableMemory rds.memory.free
SwapUsage rds.memory.swap_cached
NetworkReceiveThroughput rds.network.receive
NetworkTransmitThroughput rds.network.transmit
BinLogDiskUsage rds.binlogdiskusage.binlogdiskusage
DatabaseConnections rds.database.connections
ReadIOPS rds.diskiops.reads
WriteIOPS rds.diskiops.write
DiskQueueDepth rds.diskqueue.depth
FreeStorageSpace rds.disk.freestoragespace
ReplicaLag rds.replicalag.replicalag
ReadLatency rds.disklatency.read
WriteLatency rds.disklatency.write
ReadThroughput rds.diskthroughput.read
WriteThroughput rds.diskthroughput.write
*/

// http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/rds-metricscollected.html
var rdsGraphdefs = map[string](Graphs){
	"rds.cpu": Graphs{
		Label: "RDS CPU Utilization",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "CPUUtilization", Label: "CPU Utilization"},
		},
	},
	"rds.cpucredit": Graphs{
		Label: "RDS CPU Credit",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "CPUCreditUsage", Label: "Usage"},
			Metrics{Name: "CPUCreditBalance", Label: "Balance"},
		},
	},
	"rds.memory": Graphs{
		Label: "RDS Memory",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "FreeableMemory", Label: "Free"},
			Metrics{Name: "SwapUsage", Label: "Swap Usage"},
		},
	},
	"rds.network": Graphs{
		Label: "RDS Network",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "NetworkReceiveThroughput", Label: "Receive"},
			Metrics{Name: "NetworkTransmitThroughput", Label: "Transmit"},
		},
	},
	"rds.binlogdiskusage": Graphs{
		Label: "RDS BinLog Disk Usage",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "BinLogDiskUsage", Label: "Usage"},
		},
	},
	"rds.databaseconnections": Graphs{
		Label: "RDS Connections",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "DatabaseConnections", Label: "Connections"},
		},
	},
	"rds.diskiops": Graphs{
		Label: "RDS Disk IOPS",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "ReadIOPS", Label: "Read"},
			Metrics{Name: "WriteIOPS", Label: "Write"},
		},
	},
	"rds.diskqueue": Graphs{
		Label: "RDS Disk Queue",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "DiskQueueDepth", Label: "Depth"},
		},
	},
	"rds.disk": Graphs{
		Label: "RDS Free Storage Space",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "FreeStorageSpace", Label: "Free Space"},
		},
	},
	"rds.replicalag": Graphs{
		Label: "RDS Replica Lag",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "ReplicaLag", Label: "Lag"},
		},
	},
	"rds.disklatency": Graphs{
		Label: "RDS Disk Latency",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "ReadLatency", Label: "Read"},
			Metrics{Name: "WriteLatency", Label: "Write"},
		},
	},
	"rds.diskthrouput": Graphs{
		Label: "RDS Disk Throughput",
		Unit:  "integer",
		Metrics: [](Metrics){
			Metrics{Name: "ReadThroughput", Label: "Read"},
			Metrics{Name: "WriteThroughput", Label: "Write"},
		},
	},
}

func (s *AWSSession) getMetricStatistics(rds *AWSElement, graphdefs map[string](Graphs), namespace string, dimensions []*cloudwatch.Dimension) []*mkr.MetricValue {
	svc := cloudwatch.New(s.Sess)

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
				Namespace:  aws.String(namespace),
				Period:     aws.Int64(60),
				StartTime:  aws.Time(time.Now().Add(-2 * time.Minute)),
				Statistics: []*string{
					aws.String(statistics),
				},
				Dimensions: dimensions,
			}
			resp, err := svc.GetMetricStatistics(params)

			if err != nil {
				fmt.Println(err.Error())
				return metricValues
			}

			//fmt.Println(rds.Name, metrics.Name)
			//fmt.Println(resp)
			latestTime := int64(0)
			var latestValue float64
			for _, dp := range resp.Datapoints {
				timestamp := dp.Timestamp.Unix()
				var value float64
				switch statistics {
				case "Sum":
					value = *(dp.Sum)
				default:
					value = *(dp.Average)
				}
				if latestTime < timestamp {
					latestValue = value
					latestTime = timestamp
				}
				//fmt.Println(rds.Name, metrics.Name, value)
			}
			if latestTime > 0 {
				metricValues = append(metricValues, &mkr.MetricValue{
					Name:  "custom." + key + "." + metrics.Name,
					Value: latestValue,
					Time:  latestTime,
				})
			}
		}
	}
	return metricValues
}

func (s *AWSSession) updateAWSElementList(elbs []*AWSElement, client *mkr.Client) {
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
	return
}

func (s *AWSSession) crawlELBMetrics(client *mkr.Client, elbs []*AWSElement) {
	for _, elb := range elbs {
		dimensions := []*cloudwatch.Dimension{
			{
				Name:  aws.String("LoadBalancerName"),
				Value: aws.String(elb.Name),
			}}
		metricValues := s.getMetricStatistics(elb, elbGraphdefs, "AWS/ELB", dimensions)
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

func (s *AWSSession) crawlRDSMetrics(client *mkr.Client, rdss []*AWSElement) {
	for _, rds := range rdss {
		dimensions := []*cloudwatch.Dimension{
			{
				Name:  aws.String("DBInstanceIdentifier"),
				Value: aws.String(rds.Name),
			}}
		metricValues := s.getMetricStatistics(rds, rdsGraphdefs, "AWS/RDS", dimensions)
		logger.Log("info", fmt.Sprintf("%s", metricValues))
		err := client.PostHostMetricValuesByHostID(rds.HostID, metricValues)
		//logger.DieIf(err)
		if err != nil {
			logger.Log("error", err.Error())
		}

		for _, metric := range metricValues {
			logger.Log("thrown", fmt.Sprintf("%s '%s\t%f\t%d'", rds.HostID, metric.Name, metric.Value, metric.Time))
		}
	}
}
