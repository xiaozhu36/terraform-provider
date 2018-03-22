package alicloud

import (
	"github.com/denverdino/aliyungo/slb"
	"strings"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aws/aws-sdk-go/aws/request"
)

type Listener struct {
	slb.HTTPListenerType

	InstancePort     int
	LoadBalancerPort int
	Protocol         string
	//tcp & udp
	PersistenceTimeout int

	//https
	SSLCertificateId string

	//tcp
	HealthCheckType slb.HealthCheckType

	//api interface: http & https is HealthCheckTimeout, tcp & udp is HealthCheckConnectTimeout
	HealthCheckConnectTimeout int
}

type ListenerErr struct {
	ErrType string
	Err     error
}

func (e *ListenerErr) Error() string {
	return e.ErrType + " " + e.Err.Error()

}

type SchedulerType string

const (
	WRRScheduler = SchedulerType("wrr")
	WLCScheduler = SchedulerType("wlc")
)

type FlagType string

const (
	OnFlag  = FlagType("on")
	OffFlag = FlagType("off")
)

type StickySessionType string

const (
	InsertStickySessionType = StickySessionType("insert")
	ServerStickySessionType = StickySessionType("server")
)

const BackendServerPort = -520

type HealthCheckHttpCodeType string

const (
	HTTP_2XX = HealthCheckHttpCodeType("http_2xx")
	HTTP_3XX = HealthCheckHttpCodeType("http_3xx")
	HTTP_4XX = HealthCheckHttpCodeType("http_4xx")
	HTTP_5XX = HealthCheckHttpCodeType("http_5xx")
)

type HealthCheckType string

const (
	TCPHealthCheckType  = HealthCheckType("tcp")
	HTTPHealthCheckType = HealthCheckType("http")
)

func expandBackendServers(list []interface{}, weight int) string {
	if len(list) <= 0 {
		return ""
	}
	var servers []string
	for _, i := range list {
		if Trim(i.(string)) != "" {
			str := fmt.Sprintf("{'ServerId':'%s','Weight':'%d'}", Trim(i.(string)), weight)
			servers = append(servers, str)
		}
	}
	return fmt.Sprintf("[%s]", strings.Join(servers, COMMA_SEPARATED))
}

func getSlbCommonRequest() *requests.CommonRequest {
	req := requests.NewCommonRequest()
	req.Domain = "slb.aliyuncs.com"
	req.Version = "2014-05-26"
	return req
}

func getSlbInstanceRequest(slbId string) *requests.CommonRequest {
	req := getSlbCommonRequest()
	req.QueryParams["LoadBalancerId"] = slbId
	return req
}

func getSlbListenerRequest(slbId , port string) *requests.CommonRequest {
	req:= getSlbInstanceRequest(slbId)
	req.QueryParams["ListenerPort"] = port
	return req
}