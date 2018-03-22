package alicloud

import (
	"fmt"

	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"strconv"
)

func (client *AliyunClient) DescribeLoadBalancerAttribute(slbId string) (lb slb.DescribeLoadBalancerAttributeResponse, err error) {

	request := slb.CreateDescribeLoadBalancerAttributeRequest()
	request.RegionId = string(client.Region)
	request.LoadBalancerId = string(slbId)

	loadBalancer, err := client.slbconn.DescribeLoadBalancerAttribute(request)

	if err != nil {
		if IsExceptedError(err, LoadBalancerNotFound) {
			return lb, GetNotFoundErrorFromString(GetNotFoundMessage("Load Balancer", slbId))
		}
		return
	}

	if loadBalancer == nil || loadBalancer.LoadBalancerId != slbId {
		return lb, GetNotFoundErrorFromString(GetNotFoundMessage("Load Balancer", slbId))
	}

	return *loadBalancer, nil
}

func (client *AliyunClient) DescribeLoadBalancerListenerAttribute(slbId, listenerPort string, procotol Protocol) (response responses.CommonResponse, err error) {

	request := getSlbCommonRequest()
	request.RegionId = string(client.Region)
	request.ApiName = fmt.Sprintf("DescribeLoadBalancer%sListenerAttribute", strings.ToUpper(string(procotol)))
	request.QueryParams["ListenerPort"] = listenerPort

	response, err = client.slbconn.ProcessCommonRequest(request)

	if err != nil {
		if IsExceptedError(err, LoadBalancerNotFound) {
			return *response, GetNotFoundErrorFromString(GetNotFoundMessage("Load Balancer", slbId))
		}
		return *response, err
	}

	if response == nil || response["ListenerPort"] != listenerPort {
		return *response, GetNotFoundErrorFromString(GetNotFoundMessage("Load Balancer", slbId))
	}

	return *response, nil
}

func (client *AliyunClient) DescribeLoadBalancerRuleId(slbId string, port int, domain, url string) (string, error) {

	if rules, err := client.slbconn.DescribeRules(&slb.DescribeRulesArgs{
		RegionId:       client.Region,
		LoadBalancerId: slbId,
		ListenerPort:   port,
	}); err != nil {
		return "", fmt.Errorf("DescribeRules got an error: %#v", err)
	} else {
		for _, rule := range rules.Rules.Rule {
			if rule.Domain == domain && rule.Url == url {
				return rule.RuleId, nil
			}
		}
	}
	return "", GetNotFoundErrorFromString(fmt.Sprintf("Rule is not found based on domain %s and url %s.", domain, url))
}

func (client *AliyunClient) WaitForLoadBalancer(loadBalancerId string, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	st := strings.ToLower(string(status))
	for {
		lb, err := client.DescribeLoadBalancerAttribute(loadBalancerId)

		if err != nil && !NotFoundError(err) {
			return err
		}

		if lb.LoadBalancerStatus == st {
			//TODO
			break
		}
		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return GetTimeErrorFromString(GetTimeoutMessage("Load Balancer", string(status)))
		}
		time.Sleep(DefaultIntervalShort * time.Second)
	}
	return nil
}

func (client *AliyunClient) WaitForLoadBalancerListener(loadBalancerId string, port int, protocol Protocol, status Status, timeout int) error {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	for {
		listener, err := client.DescribeLoadBalancerListenerAttribute(loadBalancerId, strconv.Itoa(port), protocol)
		if err != nil {
			if NotFoundError(err){
				continue
			}
			return err
		}

		if strings.ToUpper(listener["Status"]) == strings.ToLower(string(status)) {
			break
		}

		timeout = timeout - DefaultIntervalShort
		if timeout <= 0 {
			return GetTimeErrorFromString(GetTimeoutMessage("Load Balancer Listener", string(status)))
		}
		time.Sleep(DefaultIntervalShort * time.Second)

	}
	return nil
}
