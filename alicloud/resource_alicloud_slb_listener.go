package alicloud

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/hashicorp/terraform/config"
)

func resourceAliyunSlbListener() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunSlbListenerCreate,
		Read:   resourceAliyunSlbListenerRead,
		Update: resourceAliyunSlbListenerUpdate,
		Delete: resourceAliyunSlbListenerDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"load_balancer_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"frontend_port": &schema.Schema{
				Type:         schema.TypeInt,
				ValidateFunc: validateInstancePort,
				Required:     true,
				ForceNew:     true,
			},
			"lb_port": &schema.Schema{
				Type:       schema.TypeInt,
				Optional:   true,
				Deprecated: "Field 'lb_port' has been deprecated, and using 'frontend_port' to replace.",
			},

			"backend_port": &schema.Schema{
				Type:         schema.TypeInt,
				ValidateFunc: validateInstancePort,
				Required:     true,
				ForceNew:     true,
			},

			"instance_port": &schema.Schema{
				Type:       schema.TypeInt,
				Optional:   true,
				Deprecated: "Field 'instance_port' has been deprecated, and using 'backend_port' to replace.",
			},

			"lb_protocol": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Field 'lb_protocol' has been deprecated, and using 'protocol' to replace.",
			},

			"protocol": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateInstanceProtocol,
				Required:     true,
				ForceNew:     true,
			},

			"bandwidth": &schema.Schema{
				Type:         schema.TypeInt,
				ValidateFunc: validateSlbListenerBandwidth,
				Required:     true,
			},
			"scheduler": &schema.Schema{
				Type:         schema.TypeString,
				ValidateFunc: validateSlbListenerScheduler,
				Optional:     true,
				Default:      WRRScheduler,
			},
			"server_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			//http & https
			"sticky_session": &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{
					string(OnFlag),
					string(OffFlag)}),
				Optional:         true,
				Default:          OffFlag,
				DiffSuppressFunc: httpHttpsDiffSuppressFunc,
			},
			//http & https
			"sticky_session_type": &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{
					string(InsertStickySessionType),
					string(ServerStickySessionType)}),
				Optional:         true,
				DiffSuppressFunc: stickySessionTypeDiffSuppressFunc,
			},
			//http & https
			"cookie_timeout": &schema.Schema{
				Type:             schema.TypeInt,
				ValidateFunc:     validateSlbListenerCookieTimeout,
				Optional:         true,
				DiffSuppressFunc: cookieTimeoutDiffSuppressFunc,
			},
			//http & https
			"cookie": &schema.Schema{
				Type:             schema.TypeString,
				ValidateFunc:     validateSlbListenerCookie,
				Optional:         true,
				DiffSuppressFunc: cookieDiffSuppressFunc,
			},
			//tcp & udp
			"persistence_timeout": &schema.Schema{
				Type:             schema.TypeInt,
				ValidateFunc:     validateSlbListenerPersistenceTimeout,
				Optional:         true,
				Default:          0,
				DiffSuppressFunc: tcpUdpDiffSuppressFunc,
			},
			//http & https
			"health_check": &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{
					string(OnFlag),
					string(OffFlag)}),
				Optional:         true,
				Default:          OnFlag,
				DiffSuppressFunc: httpHttpsDiffSuppressFunc,
			},
			//tcp
			"health_check_type": &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validateAllowedStringValue([]string{
					string(TCPHealthCheckType),
					string(HTTPHealthCheckType)}),
				Optional:         true,
				Default:          TCPHealthCheckType,
				DiffSuppressFunc: healthCheckTypeDiffSuppressFunc,
			},
			//http & https & tcp
			"health_check_domain": &schema.Schema{
				Type:             schema.TypeString,
				ValidateFunc:     validateSlbListenerHealthCheckDomain,
				Optional:         true,
				DiffSuppressFunc: httpHttpsTcpDiffSuppressFunc,
			},
			//http & https & tcp
			"health_check_uri": &schema.Schema{
				Type:             schema.TypeString,
				ValidateFunc:     validateSlbListenerHealthCheckUri,
				Optional:         true,
				Default:          "/",
				DiffSuppressFunc: httpHttpsTcpDiffSuppressFunc,
			},
			"health_check_connect_port": &schema.Schema{
				Type:             schema.TypeInt,
				ValidateFunc:     validateSlbListenerHealthCheckConnectPort,
				Optional:         true,
				Computed:         true,
				DiffSuppressFunc: healthCheckDiffSuppressFunc,
			},
			"healthy_threshold": &schema.Schema{
				Type:             schema.TypeInt,
				ValidateFunc:     validateIntegerInRange(1, 10),
				Optional:         true,
				Default:          3,
				DiffSuppressFunc: healthCheckDiffSuppressFunc,
			},
			"unhealthy_threshold": &schema.Schema{
				Type:             schema.TypeInt,
				ValidateFunc:     validateIntegerInRange(1, 10),
				Optional:         true,
				Default:          3,
				DiffSuppressFunc: healthCheckDiffSuppressFunc,
			},

			"health_check_timeout": &schema.Schema{
				Type:             schema.TypeInt,
				ValidateFunc:     validateIntegerInRange(1, 300),
				Optional:         true,
				Default:          5,
				DiffSuppressFunc: healthCheckDiffSuppressFunc,
			},
			"health_check_interval": &schema.Schema{
				Type:             schema.TypeInt,
				ValidateFunc:     validateIntegerInRange(1, 50),
				Optional:         true,
				Default:          2,
				DiffSuppressFunc: healthCheckDiffSuppressFunc,
			},
			//http & https & tcp
			"health_check_http_code": &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validateAllowedSplitStringValue([]string{
					string(HTTP_2XX),
					string(HTTP_3XX),
					string(HTTP_4XX),
					string(HTTP_5XX)}, ","),
				Optional:         true,
				Default:          HTTP_2XX,
				DiffSuppressFunc: httpHttpsTcpDiffSuppressFunc,
			},
			//https
			"ssl_certificate_id": &schema.Schema{
				Type:             schema.TypeString,
				Optional:         true,
				DiffSuppressFunc: sslCertificateIdDiffSuppressFunc,
			},
		},
	}
}

func resourceAliyunSlbListenerCreate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)

	protocol := d.Get("protocol").(string)
	lb_id := d.Get("load_balancer_id").(string)
	frontend := d.Get("frontend_port").(int)
	var err error

	switch Protocol(protocol) {
	case Https:
		ssl_id, ok := d.GetOk("ssl_certificate_id")
		if !ok || ssl_id == "" {
			return fmt.Errorf("'ssl_certificate_id': required field is not set when the protocol is 'https'.")
		}

		request, e := buildHttpListenerRequest(d)
		if e != nil {
			return err
		}
		//args := slb.CreateLoadBalancerHTTPSListenerArgs(slb.HTTPSListenerType{
		//	HTTPListenerType:    httpType,
		//	ServerCertificateId: ssl_id.(string),
		//})
		request.QueryParams["ServerCertificateId"] = ssl_id.(string)
		request.ApiName = "CreateLoadBalancerHTTPSListener"
		//err = slbconn.CreateLoadBalancerHTTPSListener(request)
		if _, err = client.slbconn.ProcessCommonRequest(request); err != nil {
			return fmt.Errorf("[ERROR] %s got an error: %#v", request.ApiName, err)
		}
	case Http:
		request, e := buildHttpListenerRequest(d)
		if e != nil {
			return e
		}

		request.ApiName = "CreateLoadBalancerHTTPListener"
		if _, err = client.slbconn.ProcessCommonRequest(request); err != nil {
			return fmt.Errorf("[ERROR] %s got an error: %#v", request.ApiName, err)
		}
	default:

		request := buildListenerCommonRequest(d)
		request.ApiName = fmt.Sprintf("CreateLoadBalancer%sListener", strings.ToUpper(protocol))
		if _, err = client.slbconn.ProcessCommonRequest(request); err != nil {
			return fmt.Errorf("[ERROR] %s got an error: %#v", request.ApiName, err)
		}
	}

	if err != nil {
		if IsExceptedError(err, ListenerAlreadyExists) {
			return fmt.Errorf("The listener with the frontend port %d already exists. Please define a new 'alicloud_slb_listener' resource and "+
				"use ID '%s:%d' to import it or modify its frontend port and then try again.", frontend, lb_id, frontend)
		}
		return fmt.Errorf("Create %s Listener got an error: %#v", protocol, err)
	}

	d.SetId(lb_id + ":" + strconv.Itoa(frontend))

	if err := client.WaitForLoadBalancerListener(lb_id, frontend, Protocol(protocol), Stopped, DefaultTimeout); err != nil {
		return fmt.Errorf("WaitForListener %s got error: %#v", Stopped, err)
	}

	request := getSlbCommonRequest()
	request.RegionId = string(getRegion(d, meta))
	request.ApiName = "StartLoadBalancerListener"
	request.QueryParams["LoadBalancerId"] = lb_id
	request.QueryParams["ListenerPort"] = frontend
	if _, err := client.slbconn.ProcessCommonRequest(request); err != nil {
		return err
	}

	if err := client.WaitForLoadBalancerListener(lb_id, frontend, Protocol(protocol), Running, DefaultTimeout); err != nil {
		return fmt.Errorf("WaitForListener %s got error: %#v", Running, err)
	}

	return resourceAliyunSlbListenerUpdate(d, meta)
}

func resourceAliyunSlbListenerRead(d *schema.ResourceData, meta interface{}) error {
	lb_id, protocol, port, err := parseListenerId(d, meta)
	if err != nil {
		return fmt.Errorf("Get slb listener got an error: %#v", err)
	}

	if protocol == "" {
		d.SetId("")
		return nil
	}
	d.Set("protocol", protocol)
	d.Set("load_balancer_id", lb_id)

	listener, err := meta.(*AliyunClient).DescribeLoadBalancerListenerAttribute(lb_id, port, Protocol(protocol))

	if err != nil {
		if IsExceptedError(err, ListenerNotFound) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("DescribeLoadBalancer%sListenerAttribute got an error: %#v", strings.ToUpper(protocol), err)
	}

	readListener(d, listener)
	return nil

	//switch Protocol(protocol) {
	//case Https:
	//	//https_ls, err := slbconn.DescribeLoadBalancerHTTPSListenerAttribute(lb_id, port)
	//	return readListenerAttribute(d, protocol, listener, err)
	//case Tcp:
	//	//tcp_ls, err := slbconn.DescribeLoadBalancerTCPListenerAttribute(lb_id, port)
	//	return readListenerAttribute(d, protocol, listener, err)
	//case Udp:
	//	//udp_ls, err := slbconn.DescribeLoadBalancerUDPListenerAttribute(lb_id, port)
	//	return readListenerAttribute(d, protocol, listener, err)
	//default:
	//	//http_ls, err := slbconn.DescribeLoadBalancerHTTPListenerAttribute(lb_id, port)
	//	return readListenerAttribute(d, protocol, listener, err)
	//}
}

func resourceAliyunSlbListenerUpdate(d *schema.ResourceData, meta interface{}) error {

	slbconn := meta.(*AliyunClient).slbconn
	protocol := Protocol(d.Get("protocol").(string))

	d.Partial(true)

	request := buildListenerCommonRequest(d)
	//httpType, err := buildHttpListenerType(d)
	//if (protocol == Https || protocol == Http) && err != nil {
	//	return err
	//}
	//tcpArgs := slb.SetLoadBalancerTCPListenerAttributeArgs(buildTcpListenerArgs(d))
	//udpArgs := slb.SetLoadBalancerUDPListenerAttributeArgs(buildUdpListenerArgs(d))
	//httpsArgs := slb.SetLoadBalancerHTTPSListenerAttributeArgs(slb.CreateLoadBalancerHTTPSListenerArgs(slb.HTTPSListenerType{}))

	update := false
	if d.HasChange("scheduler") {
		request.QueryParams["Scheduler"] = d.Get("scheduler").(string)
		//httpType.Scheduler = scheduler
		//tcpArgs.Scheduler = scheduler
		//udpArgs.Scheduler = scheduler
		d.SetPartial("scheduler")
		update = true
	}

	if d.HasChange("server_group_id") {
		request.QueryParams["VServerGroupId"] = d.Get("server_group_id").(string)
		//httpType.VServerGroupId = groupId
		//tcpArgs.VServerGroupId = groupId
		//udpArgs.VServerGroupId = groupId
		d.SetPartial("server_group_id")
		update = true
	}

	// http https
		if d.HasChange("sticky_session") {
			request.QueryParams["StickySession"] = d.Get("sticky_session").(string)
			d.SetPartial("sticky_session")
			update = true
		}
		if d.HasChange("sticky_session_type") {
			request.QueryParams["StickySessionType"] = d.Get("sticky_session_type").(string)
			d.SetPartial("sticky_session_type")
			update = true
		}
		if d.HasChange("cookie_timeout") {
			request.QueryParams["CookieTimeout"] = requests.NewInteger(d.Get("cookie_timeout").(int))
			d.SetPartial("cookie_timeout")
			update = true
		}
		if d.HasChange("cookie") {
			request.QueryParams["Cookie"] = d.Get("cookie").(string)
			d.SetPartial("cookie")
			update = true
		}

		if d.HasChange("health_check") {
			request.QueryParams["HealthCheck"] = d.Get("health_check").(string)
			d.SetPartial("health_check")
			update = true
		}

	// http https tcp
		if d.HasChange("health_check_domain") {
			if domain, ok := d.GetOk("health_check_domain"); ok {
				request.QueryParams["HealthCheckDomain"] = domain.(string)
				//httpType.HealthCheckDomain = domain.(string)
				//tcpArgs.HealthCheckDomain = domain.(string)
				d.SetPartial("health_check_domain")
				update = true
			}
		}
		if d.HasChange("health_check_uri") {
			request.QueryParams["HealthCheckURI"] = d.Get("health_check_uri").(string)
			d.SetPartial("health_check_uri")
			update = true
		}
		if d.HasChange("health_check_http_code") {
			request.QueryParams["HealthCheckHttpCode"] = d.Get("health_check_http_code").(string)
			d.SetPartial("health_check_http_code")
			update = true
		}

	// http https tcp udp and health_check=on
	if d.HasChange("unhealthy_threshold") {
		request.QueryParams["UnhealthyThreshold"] = requests.NewInteger(d.Get("unhealthy_threshold").(int))
		//udpArgs.UnhealthyThreshold = d.Get("unhealthy_threshold").(int)
		d.SetPartial("unhealthy_threshold")
		update = true
		//}
	}
	if d.HasChange("healthy_threshold") {
		request.QueryParams["HealthyThreshold"] = requests.NewInteger(d.Get("healthy_threshold").(int))
		//udpArgs.HealthyThreshold = d.Get("healthy_threshold").(int)
		d.SetPartial("healthy_threshold")
		update = true
	}
	if d.HasChange("health_check_timeout") {
		request.QueryParams["HealthCheckConnectTimeout"] =requests.NewInteger(d.Get("health_check_timeout").(int))
		//udpArgs.HealthCheckConnectTimeout = d.Get("health_check_timeout").(int)
		d.SetPartial("health_check_timeout")
		update = true
	}
	if d.HasChange("health_check_interval") {
		request.QueryParams["HealthCheckInterval"] = requests.NewInteger(d.Get("health_check_interval").(int))
		//udpArgs.HealthCheckInterval = d.Get("health_check_interval").(int)
		d.SetPartial("health_check_interval")
		update = true
	}
	if d.HasChange("health_check_connect_port") {
		if port, ok := d.GetOk("health_check_connect_port"); ok {
			request.QueryParams["HealthCheckConnectPort"] = requests.NewInteger(port.(int))
			//tcpArgs.HealthCheckConnectPort = port.(int)
			//udpArgs.HealthCheckConnectPort = port.(int)
			d.SetPartial("health_check_connect_port")
			update = true
		}
	}

	// tcp and udp
	if d.HasChange("persistence_timeout") {
		request.QueryParams["PersistenceTimeout"] = requests.NewInteger(d.Get("persistence_timeout").(int))
		//udpArgs.PersistenceTimeout = d.Get("persistence_timeout").(int)
		d.SetPartial("persistence_timeout")
		update = true
	}

	// tcp
	if d.HasChange("health_check_type") {
		request.QueryParams["HealthCheckType"] = d.Get("health_check_type").(string)
		d.SetPartial("health_check_type")
		update = true
	}

	// https
	if protocol == Https {
		ssl_id, ok := d.GetOk("ssl_certificate_id")
		if !ok && ssl_id == "" {
			return fmt.Errorf("'ssl_certificate_id': required field is not set when the protocol is 'https'.")
		}

		request.QueryParams["ServerCertificateId"] = ssl_id.(string)
		if d.HasChange("ssl_certificate_id") {
			d.SetPartial("ssl_certificate_id")
			update = true
		}
	}

	if update {
		request.ApiName = fmt.Sprintf("SetLoadBalancer%sListenerAttribute", strings.ToUpper(string(protocol)))
		if _, err := slbconn.ProcessCommonRequest(request); err != nil {
			return fmt.Errorf("[ERROR] %s got an error: %#v", request.ApiName, err)
		}
		//switch protocol {
		//case Https:
		//	httpsArgs.HTTPListenerType = httpType
		//	if err := slbconn.SetLoadBalancerHTTPSListenerAttribute(&httpsArgs); err != nil {
		//		return fmt.Errorf("SetHTTPSListenerAttribute got an error: %#v", err)
		//	}
		//case Tcp:
		//	if err := slbconn.SetLoadBalancerTCPListenerAttribute(&tcpArgs); err != nil {
		//		return fmt.Errorf("SetTCPListenerAttribute got an error: %#v", err)
		//	}
		//case Udp:
		//	if err := slbconn.SetLoadBalancerUDPListenerAttribute(&udpArgs); err != nil {
		//		return fmt.Errorf("SetTCPListenerAttribute got an error: %#v", err)
		//	}
		//default:
		//	httpArgs := slb.SetLoadBalancerHTTPListenerAttributeArgs(slb.CreateLoadBalancerHTTPListenerArgs(httpType))
		//	if err := slbconn.SetLoadBalancerHTTPListenerAttribute(&httpArgs); err != nil {
		//		return fmt.Errorf("SetHTTPListenerAttribute got an error: %#v", err)
		//	}
		//}
	}

	d.Partial(false)

	return resourceAliyunSlbListenerRead(d, meta)
}

func resourceAliyunSlbListenerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	lb_id, protocol, port, err := parseListenerId(d, meta)
	if err != nil {
		return fmt.Errorf("Get slb listener got an error: %#v", err)
	}

	if protocol == "" {
		d.SetId("")
		return nil
	}
	req := getSlbListenerRequest(lb_id, port)
	req.ApiName="DeleteLoadBalancerListener"

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := client.slbconn.ProcessCommonRequest(req)

		if err != nil {
			if IsExceptedError(err, SystemBusy) {
				resource.RetryableError(fmt.Errorf("Delete load balancer listener timeout and got an error: %#v.", err))
			}
			return resource.NonRetryableError(err)
		}

		if _, err := client.DescribeLoadBalancerListenerAttribute(lb_id, port, Protocol(protocol)); err != nil {
			if IsExceptedError(err, ListenerNotFound) {
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("DescribeLoadBalancer%sListenerAttribute got an error: %#v", strings.ToUpper(protocol), err))
		}
		return nil
	})
}

func buildListenerCommonRequest(d *schema.ResourceData) *requests.CommonRequest {

	request := getSlbCommonRequest()
	request.QueryParams["LoadBalancerId"] = d.Get("load_balancer_id").(string)
	request.QueryParams["ListenerPort"] = requests.NewInteger(d.Get("frontend_port").(int))
	request.QueryParams["BackendServerPort"] = requests.NewInteger(d.Get("backend_port").(int))
	request.QueryParams["Bandwidth"] = requests.NewInteger(d.Get("bandwidth").(int))
	request.QueryParams["VServerGroupId"]=    d.Get("server_group_id").(string)

	return request

}
func buildHttpListenerRequest(d *schema.ResourceData) (*requests.CommonRequest, error) {

	request := buildListenerCommonRequest(d)
	stickySession := d.Get("sticky_session").(string)
	healthCheck := d.Get("health_check").(string)
	request.QueryParams["StickySession"] = stickySession
	request.QueryParams["HealthCheck"] = healthCheck

	if stickySession == string(OnFlag) {
		sessionType, ok := d.GetOk("sticky_session_type")
		if !ok || sessionType.(string) == "" {
			return nil, fmt.Errorf("'sticky_session_type': required field is not set when the StickySession is %s.", OnFlag)
		} else {
			request.QueryParams["StickySessionType"] = sessionType.(string)

		}
		if sessionType == string(InsertStickySessionType) {
			if timeout, ok := d.GetOk("cookie_timeout"); !ok || timeout == 0 {
				return nil, fmt.Errorf("'cookie_timeout': required field is not set when the StickySession is %s "+
					"and StickySessionType is %s.", OnFlag, InsertStickySessionType)
			} else {
				request.QueryParams["CookieTimeout"] = requests.NewInteger(timeout.(int))
			}
		} else {
			if cookie, ok := d.GetOk("cookie"); !ok || cookie.(string) == "" {
				return nil, fmt.Errorf("'cookie': required field is not set when the StickySession is %s "+
					"and StickySessionType is %s.", OnFlag, ServerStickySessionType)
			} else {
				request.QueryParams["Cookie"] = cookie.(string)
			}
		}
	}
	if healthCheck == string(OnFlag) {
		request.QueryParams["HealthCheckURI"] = d.Get("health_check_uri").(string)
		if port, ok := d.GetOk("health_check_connect_port"); !ok || port.(int) == 0 {
			return nil, fmt.Errorf("'health_check_connect_port': required field is not set when the HealthCheck is %s.", OnFlag)
		} else {
			request.QueryParams["HealthCheckConnectPort"] = requests.NewInteger(port.(int))
		}
		request.QueryParams["HealthyThreshold"] = requests.NewInteger(d.Get("healthy_threshold").(int))
		request.QueryParams["UnhealthyThreshold"] = requests.NewInteger(d.Get("unhealthy_threshold").(int))
		request.QueryParams["HealthCheckTimeout"] = requests.NewInteger(d.Get("health_check_timeout").(int))
		request.QueryParams["HealthCheckInterval"] = requests.NewInteger(d.Get("health_check_interval").(int))
		request.QueryParams["HealthCheckHttpCode"] = d.Get("health_check_http_code").(string)
	}
	return request, nil
}

func checkHttpListenerParams(d *schema.ResourceData) error {
	stickySession := d.Get("sticky_session").(string)
	healthCheck := d.Get("health_check").(string)
	if stickySession == string(OnFlag) {
		sessionType, ok := d.GetOk("sticky_session_type")
		if !ok || sessionType.(string) == "" {
			return fmt.Errorf("'sticky_session_type': required field is not set when the StickySession is %s.", OnFlag)
		}

		if sessionType == string(InsertStickySessionType) {
			if timeout, ok := d.GetOk("cookie_timeout"); !ok || timeout == 0 {
				return fmt.Errorf("'cookie_timeout': required field is not set when the StickySession is %s "+
					"and StickySessionType is %s.", OnFlag, InsertStickySessionType)
			}
		} else {
			if cookie, ok := d.GetOk("cookie"); !ok || cookie.(string) == "" {
				return fmt.Errorf("'cookie': required field is not set when the StickySession is %s "+
					"and StickySessionType is %s.", OnFlag, ServerStickySessionType)
			}
		}
	}
	if healthCheck == string(OnFlag) {
		if port, ok := d.GetOk("health_check_connect_port"); !ok || port.(int) == 0 {
			return fmt.Errorf("'health_check_connect_port': required field is not set when the HealthCheck is %s.", OnFlag)
		}
	}
	return nil
}

//func buildTcpListenerArgs(d *schema.ResourceData) slb.CreateLoadBalancerTCPListenerArgs {
//
//	return slb.CreateLoadBalancerTCPListenerArgs(slb.TCPListenerType{
//		LoadBalancerId:    d.Get("load_balancer_id").(string),
//		ListenerPort:      d.Get("frontend_port").(int),
//		BackendServerPort: d.Get("backend_port").(int),
//		Bandwidth:         d.Get("bandwidth").(int),
//		VServerGroupId:    d.Get("server_group_id").(string),
//	})
//}
//func buildUdpListenerArgs(d *schema.ResourceData) slb.CreateLoadBalancerUDPListenerArgs {
//
//	return slb.CreateLoadBalancerUDPListenerArgs(slb.UDPListenerType{
//		LoadBalancerId:    d.Get("load_balancer_id").(string),
//		ListenerPort:      d.Get("frontend_port").(int),
//		BackendServerPort: d.Get("backend_port").(int),
//		Bandwidth:         d.Get("bandwidth").(int),
//		VServerGroupId:    d.Get("server_group_id").(string),
//	})
//}

func parseListenerId(d *schema.ResourceData, meta interface{}) (string, string,string, error) {
	slbconn := meta.(*AliyunClient).slbconn
	parts := strings.Split(d.Id(), ":")
	port := parts[1]

	loadBalancer, err := slbconn.DescribeLoadBalancerAttribute(parts[0])
	if err != nil {
		if IsExceptedError(err, LoadBalancerNotFound) {
			return "", "", "", nil
		}
		return "", "", "", fmt.Errorf("DescribeLoadBalancerAttribute got an error: %#v", err)
	}
	for _, portAndProtocol := range loadBalancer.ListenerPortsAndProtocol.ListenerPortAndProtocol {
		if requests.NewInteger(portAndProtocol.ListenerPort) == port {
			return loadBalancer.LoadBalancerId, portAndProtocol.ListenerProtocol, port, nil
		}
	}
	return "", "", "", nil
}

//func readListenerAttribute(d *schema.ResourceData, protocol string, listen interface{}, err error) error {
//	v := reflect.ValueOf(listen).Elem()
//
//	if err != nil {
//		if IsExceptedError(err, ListenerNotFound) {
//			d.SetId("")
//			return nil
//		}
//		return fmt.Errorf("DescribeLoadBalancer%sListenerAttribute got an error: %#v", strings.ToUpper(protocol), err)
//	}
//	if port := v.FieldByName("ListenerPort"); port.IsValid() && port.Interface().(int) > 0 {
//		readListener(d, listen)
//	} else {
//		d.SetId("")
//	}
//	return nil
//}

func readListener(d *schema.ResourceData, listen interface{}){
	v := reflect.ValueOf(listen).Elem()

	if val := v.FieldByName("BackendServerPort"); val.IsValid() {
		d.Set("backend_port", val.Interface().(int))
	}
	if val := v.FieldByName("ListenerPort"); val.IsValid() {
		d.Set("frontend_port", val.Interface().(int))
	}
	if val := v.FieldByName("Bandwidth"); val.IsValid() {
		d.Set("bandwidth", val.Interface().(int))
	}
	if val := v.FieldByName("Scheduler"); val.IsValid() {
		d.Set("scheduler", string(val.Interface().(string)))
	}
	if val := v.FieldByName("VServerGroupId"); val.IsValid() {
		d.Set("server_group_id", string(val.Interface().(string)))
	}
	if val := v.FieldByName("HealthCheck"); val.IsValid() {
		d.Set("health_check", string(val.Interface().(string)))
	}
	if val := v.FieldByName("StickySession"); val.IsValid() {
		d.Set("sticky_session", string(val.Interface().(string)))
	}
	if val := v.FieldByName("StickySessionType"); val.IsValid() {
		d.Set("sticky_session_type", string(val.Interface().(string)))
	}
	if val := v.FieldByName("CookieTimeout"); val.IsValid() {
		d.Set("cookie_timeout", val.Interface().(int))
	}
	if val := v.FieldByName("Cookie"); val.IsValid() {
		d.Set("cookie", val.Interface().(string))
	}
	if val := v.FieldByName("PersistenceTimeout"); val.IsValid() {
		d.Set("persistence_timeout", val.Interface().(int))
	}
	if val := v.FieldByName("HealthCheckType"); val.IsValid() {
		d.Set("health_check_type", string(val.Interface().(string)))
	}
	if val := v.FieldByName("HealthCheckDomain"); val.IsValid() {
		d.Set("health_check_domain", val.Interface().(string))
	}
	if val := v.FieldByName("HealthCheckConnectPort"); val.IsValid() {
		d.Set("health_check_connect_port", val.Interface().(int))
	}
	if val := v.FieldByName("HealthCheckURI"); val.IsValid() {
		d.Set("health_check_uri", val.Interface().(string))
	}
	if val := v.FieldByName("HealthyThreshold"); val.IsValid() {
		d.Set("healthy_threshold", val.Interface().(int))
	}
	if val := v.FieldByName("UnhealthyThreshold"); val.IsValid() {
		d.Set("unhealthy_threshold", val.Interface().(int))
	}
	if val := v.FieldByName("HealthCheckTimeout"); val.IsValid() {
		d.Set("health_check_timeout", val.Interface().(int))
	}
	if val := v.FieldByName("HealthCheckConnectTimeout"); val.IsValid() {
		d.Set("health_check_timeout", val.Interface().(int))
	}
	if val := v.FieldByName("HealthCheckInterval"); val.IsValid() {
		d.Set("health_check_interval", val.Interface().(int))
	}
	if val := v.FieldByName("HealthCheckHttpCode"); val.IsValid() {
		d.Set("health_check_http_code", string(val.Interface().(string)))
	}
	if val := v.FieldByName("ServerCertificateId"); val.IsValid() {
		d.Set("ssl_certificate_id", val.Interface().(string))
	}

	return
}

func ensureListenerAbsent(d *schema.ResourceData, protocol string, listen interface{}, err error) *resource.RetryError {
	v := reflect.ValueOf(listen).Elem()

	if err != nil {
		if IsExceptedError(err, ListenerNotFound) {
			d.SetId("")
			return nil
		}
		return resource.NonRetryableError(fmt.Errorf("While deleting listener, DescribeLoadBalancer%sListenerAttribute got an error: %#v", protocol, err))
	}
	if port := v.FieldByName("ListenerPort"); port.IsValid() && port.Interface().(int) > 0 {
		return resource.RetryableError(fmt.Errorf("Delete load balancer listener timeout and got an error: %#v.", err))
	}
	d.SetId("")
	return nil
}
