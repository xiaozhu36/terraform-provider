package alicloud

import (
	"fmt"
	"log"

	"encoding/base64"
	"encoding/json"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"time"
)

func resourceAliyunInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunInstanceCreate,
		Read:   resourceAliyunInstanceRead,
		Update: resourceAliyunInstanceUpdate,
		Delete: resourceAliyunInstanceDelete,

		Schema: map[string]*schema.Schema{
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"allocate_public_ip": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"instance_name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "ECS-Instance",
				ValidateFunc: validateInstanceName,
			},

			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateInstanceDescription,
			},

			"internet_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInternetChargeType,
			},
			"internet_max_bandwidth_in": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"internet_max_bandwidth_out": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInternetMaxBandWidthOut,
			},
			"host_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"password": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"io_optimized": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateIoOptimized,
			},

			"system_disk_category": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "cloud",
				Optional: true,
				ForceNew: true,
				ValidateFunc: validateAllowedStringValue([]string{
					string(ecs.DiskCategoryCloud),
					string(ecs.DiskCategoryCloudSSD),
					string(ecs.DiskCategoryCloudEfficiency),
					string(ecs.DiskCategoryEphemeralSSD),
				}),
			},
			"system_disk_size": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateIntegerInRange(40, 500),
			},

			//subnet_id and vswitch_id both exists, cause compatible old version, and aws habit.
			//Default vpc strategy results in new ecs instance's default network type is vpc when vswitch_id's/subnet_id's value is null.
			//In order to create classic instance, there needs to set vswitch_id's/subnet_id's value as "NIL" forcedly.
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"vswitch_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"instance_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInstanceChargeType,
			},
			"period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAliyunInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	// create postpaid instance by runInstances API
	if v := d.Get("instance_charge_type").(string); v != string(common.PrePaid) {
		return resourceAliyunRunInstance(d, meta)
	}

	args, err := buildAliyunInstanceArgs(d, meta)
	if err != nil {
		return err
	}

	instanceID, err := conn.CreateInstance(args)
	if err != nil {
		return fmt.Errorf("Error creating Aliyun ecs instance: %#v", err)
	}

	d.SetId(instanceID)

	d.Set("password", d.Get("password"))

	if err := allocateIpAndBandWidthRelative(d, meta); err != nil {
		return fmt.Errorf("allocateIpAndBandWidthRelative err: %#v", err)
	}

	// after instance created, its status is pending,
	// so we need to wait it become to stopped and then start it
	if err := conn.WaitForInstance(d.Id(), ecs.Stopped, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %#v", ecs.Stopped, err)
	}

	if err := conn.StartInstance(d.Id()); err != nil {
		return fmt.Errorf("Start instance got error: %#v", err)
	}

	if err := conn.WaitForInstance(d.Id(), ecs.Running, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %#v", ecs.Running, err)
	}

	return resourceAliyunInstanceUpdate(d, meta)
}

func resourceAliyunRunInstance(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn
	newConn := meta.(*AliyunClient).ecsNewconn

	args, err := buildAliyunInstanceArgs(d, meta)
	if err != nil {
		return err
	}

	runArgs, err := buildAliyunRunInstancesArgs(d, meta)
	if err != nil {
		return err
	}

	runArgs.CreateInstanceArgs = *args

	// runInstances is support in version 2016-03-14
	instanceIds, err := newConn.RunInstances(runArgs)

	if err != nil {
		return fmt.Errorf("Error creating Aliyun ecs instance: %#v", err)
	}

	d.SetId(instanceIds[0])

	d.Set("password", d.Get("password"))
	d.Set("system_disk_category", d.Get("system_disk_category"))
	d.Set("system_disk_size", d.Get("system_disk_size"))

	// after instance created, its status change from pending, starting to running
	if err := conn.WaitForInstanceAsyn(d.Id(), ecs.Running, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %#v", ecs.Running, err)
	}

	if err := allocateIpAndBandWidthRelative(d, meta); err != nil {
		return fmt.Errorf("allocateIpAndBandWidthRelative err: %#v", err)
	}

	if err := conn.WaitForInstanceAsyn(d.Id(), ecs.Running, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %#v", ecs.Running, err)
	}

	return resourceAliyunInstanceUpdate(d, meta)
}

func resourceAliyunInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.ecsconn

	instance, err := client.QueryInstancesById(d.Id())

	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error DescribeInstanceAttribute: %#v", err)
	}

	disk, diskErr := client.QueryInstanceSystemDisk(d.Id())

	if diskErr != nil {
		if notFoundError(diskErr) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error DescribeSystemDisk: %#v", err)
	}

	d.Set("instance_name", instance.InstanceName)
	d.Set("description", instance.Description)
	d.Set("status", instance.Status)
	d.Set("availability_zone", instance.ZoneId)
	d.Set("host_name", instance.HostName)
	d.Set("image_id", instance.ImageId)
	d.Set("instance_type", instance.InstanceType)
	d.Set("system_disk_category", disk.Category)
	d.Set("system_disk_size", disk.Size)

	if d.Get("allocate_public_ip").(bool) {
		d.Set("public_ip", instance.PublicIpAddress.IpAddress[0])
	}

	if ecs.StringOrBool(instance.IoOptimized).Value {
		d.Set("io_optimized", "optimized")
	} else {
		d.Set("io_optimized", "none")
	}

	// In Classic network, internet_charge_type is valid in any case, and its default value is 'PayByBanwidth'.
	// In VPC network, internet_charge_type is valid when instance has public ip, and its default value is 'PayByBanwidth'.
	if d.Get("subnet_id").(string) == string(NIL) || d.Get("vswitch_id").(string) == string(NIL) {
		ipAddress := strings.Join(ecs.IpAddressSetType(instance.InnerIpAddress).IpAddress, ",")
		d.Set("private_ip", ipAddress)
		d.Set("subnet_id", string(NIL))
		d.Set("vswitch_id", string(NIL))
		d.Set("internet_charge_type", instance.InternetChargeType)
	} else {
		ipAddress := instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
		d.Set("private_ip", ipAddress)
		d.Set("subnet_id", instance.VpcAttributes.VSwitchId)
		d.Set("vswitch_id", instance.VpcAttributes.VSwitchId)
		if d.Get("allocate_public_ip").(bool) {
			d.Set("internet_charge_type", instance.InternetChargeType)
		}
	}

	if d.Get("user_data").(string) != "" {
		ud, err := conn.DescribeUserdata(&ecs.DescribeUserdataArgs{
			RegionId:   getRegion(d, meta),
			InstanceId: d.Id(),
		})

		if err != nil {
			log.Printf("[ERROR] DescribeUserData for instance got error: %#v", err)
		}
		d.Set("user_data", userDataHashSum(ud.UserData))
	}

	tags, _, err := conn.DescribeTags(&ecs.DescribeTagsArgs{
		RegionId:     getRegion(d, meta),
		ResourceType: ecs.TagResourceInstance,
		ResourceId:   d.Id(),
	})

	if err != nil {
		log.Printf("[ERROR] DescribeTags for instance got error: %#v", err)
	}
	d.Set("tags", tagsToMap(tags))

	return nil
}

func resourceAliyunInstanceUpdate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)
	conn := client.ecsconn

	d.Partial(true)

	if err := setTags(client, ecs.TagResourceInstance, d); err != nil {
		log.Printf("[DEBUG] Set tags for instance got error: %#v", err)
		return fmt.Errorf("Set tags for instance got error: %#v", err)
	} else {
		d.SetPartial("tags")
	}

	attributeUpdate := false
	args := &ecs.ModifyInstanceAttributeArgs{
		InstanceId: d.Id(),
	}

	if d.HasChange("instance_name") {
		log.Printf("[DEBUG] ModifyInstanceAttribute instance_name")
		d.SetPartial("instance_name")
		args.InstanceName = d.Get("instance_name").(string)

		attributeUpdate = true
	}

	if d.HasChange("description") {
		log.Printf("[DEBUG] ModifyInstanceAttribute description")
		d.SetPartial("description")
		args.Description = d.Get("description").(string)

		attributeUpdate = true
	}

	if d.HasChange("host_name") {
		log.Printf("[DEBUG] ModifyInstanceAttribute host_name")
		d.SetPartial("host_name")
		args.HostName = d.Get("host_name").(string)

		attributeUpdate = true
	}

	passwordUpdate := false
	if d.HasChange("password") {
		log.Printf("[DEBUG] ModifyInstanceAttribute password")
		d.SetPartial("password")
		args.Password = d.Get("password").(string)

		attributeUpdate = true
		passwordUpdate = true
	}

	if attributeUpdate {
		if err := conn.ModifyInstanceAttribute(args); err != nil {
			return fmt.Errorf("Modify instance attribute got error: %#v", err)
		}
	}

	if passwordUpdate {
		if v, ok := d.GetOk("status"); ok && v.(string) != "" {
			if ecs.InstanceStatus(d.Get("status").(string)) == ecs.Running {
				log.Printf("[DEBUG] RebootInstance after change password")
				if err := conn.RebootInstance(d.Id(), false); err != nil {
					return fmt.Errorf("RebootInstance got error: %#v", err)
				}

				if err := conn.WaitForInstance(d.Id(), ecs.Running, defaultTimeout); err != nil {
					return fmt.Errorf("WaitForInstance got error: %#v", err)
				}
			}
		}
	}

	if d.HasChange("security_groups") {
		o, n := d.GetChange("security_groups")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		rl := expandStringList(os.Difference(ns).List())
		al := expandStringList(ns.Difference(os).List())

		if len(al) > 0 {
			err := client.JoinSecurityGroups(d.Id(), al)
			if err != nil {
				return err
			}
		}
		if len(rl) > 0 {
			err := client.LeaveSecurityGroups(d.Id(), rl)
			if err != nil {
				return err
			}
		}

		d.SetPartial("security_groups")
	}

	//This provider doesn't support to modify vswitch_id and subnet_id
	vsw_o, vsw_n := d.GetChange("vswitch_id")
	sub_o, sub_n := d.GetChange("subnet_id")
	if !((vsw_n == "" && !d.HasChange("subnet_id")) || (sub_n == "" && !d.HasChange("vswitch_id"))) {
		if vsw_o != "" || sub_o != "" {
			vswitchId := d.Get("vswitch_id").(string)
			subnetId := d.Get("subnet_id").(string)
			instance, err := client.QueryInstancesById(d.Id())

			if err != nil {
				return err
			}

			if instance.InstanceNetworkType == string(ClassicNet) {
				return fmt.Errorf("Update resource failed. 'vswitch_id' or 'subnet_id' isn't allowed to modify. Current instance's network type is classic.")
			}

			vswitchValue := instance.VpcAttributes.VSwitchId
			vpcValue := instance.VpcAttributes.VpcId
			vswitch, err := client.QueryVswitchById(vpcValue, vswitchValue)

			if err != nil {
				return err
			}

			if !vswitch.IsDefault || vswitchId != "" || subnetId != "" {
				return fmt.Errorf("Update resource failed. 'vswitch_id' or 'subnet_id' isn't allowed to modify. Current instance's vswitch id is %#v.", vswitchValue)
			}
		}
	}

	d.Partial(false)
	return resourceAliyunInstanceRead(d, meta)
}

func resourceAliyunInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.ecsconn

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		instance, err := client.QueryInstancesById(d.Id())
		if err != nil {
			if notFoundError(err) {
				return nil
			}
		}

		if instance.Status != ecs.Stopped {
			if err := conn.StopInstance(d.Id(), true); err != nil {
				return resource.RetryableError(fmt.Errorf("ECS stop error - trying again."))
			}

			if err := conn.WaitForInstance(d.Id(), ecs.Stopped, defaultTimeout); err != nil {
				return resource.RetryableError(fmt.Errorf("Waiting for ecs stopped timeout - trying again."))
			}
		}

		if err := conn.DeleteInstance(d.Id()); err != nil {
			return resource.RetryableError(fmt.Errorf("ECS Instance in use - trying again while it is deleted."))
		}

		return nil
	})

}

func allocateIpAndBandWidthRelative(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn
	if d.Get("allocate_public_ip").(bool) {
		if d.Get("internet_max_bandwidth_out") == 0 {
			return fmt.Errorf("Error: if allocate_public_ip is true than the internet_max_bandwidth_out cannot equal zero.")
		}
		_, err := conn.AllocatePublicIpAddress(d.Id())
		if err != nil {
			return fmt.Errorf("[DEBUG] AllocatePublicIpAddress for instance got error: %#v", err)
		}
	}
	return nil
}

func buildAliyunRunInstancesArgs(d *schema.ResourceData, meta interface{}) (*ecs.RunInstanceArgs, error) {
	args := &ecs.RunInstanceArgs{
		MaxAmount: DEFAULT_INSTANCE_COUNT,
		MinAmount: DEFAULT_INSTANCE_COUNT,
	}

	bussStr, err := json.Marshal(DefaultBusinessInfo)
	if err != nil {
		log.Printf("Failed to translate bussiness info %#v from json to string", DefaultBusinessInfo)
	}

	args.BusinessInfo = string(bussStr)
	subnetValue := d.Get("subnet_id").(string)
	vswitchValue := d.Get("vswitch_id").(string)

	// because runInstance is not compatible with createInstance, force NetworkType value to classic
	//In order to create classic instance, there needs to set instance's network type as "classic" forcedly.
	if subnetValue == string(NIL) || vswitchValue == string(NIL) {
		args.NetworkType = string(ClassicNet)
	}

	return args, nil
}

func buildAliyunInstanceArgs(d *schema.ResourceData, meta interface{}) (*ecs.CreateInstanceArgs, error) {
	client := meta.(*AliyunClient)

	args := &ecs.CreateInstanceArgs{
		RegionId:     getRegion(d, meta),
		InstanceType: d.Get("instance_type").(string),
	}

	imageID := d.Get("image_id").(string)

	args.ImageId = imageID

	systemDiskCategory := ecs.DiskCategory(d.Get("system_disk_category").(string))
	systemDiskSize := d.Get("system_disk_size").(int)

	zoneID := d.Get("availability_zone").(string)
	// check instanceType and systemDiskCategory, when zoneID is not empty
	if zoneID != "" {
		zone, err := client.DescribeZone(zoneID)
		if err != nil {
			return nil, err
		}

		if err := client.ResourceAvailable(zone, ecs.ResourceTypeInstance); err != nil {
			return nil, err
		}

		if err := client.DiskAvailable(zone, systemDiskCategory); err != nil {
			return nil, err
		}

		args.ZoneId = zoneID

	}

	args.SystemDisk = ecs.SystemDiskType{
		Category: systemDiskCategory,
		Size:     systemDiskSize,
	}

	sgs, ok := d.GetOk("security_groups")

	if ok {
		sgList := expandStringList(sgs.(*schema.Set).List())
		sg0 := sgList[0]
		// check security group instance exist
		_, err := client.DescribeSecurity(sg0)
		if err == nil {
			args.SecurityGroupId = sg0
		}
	}

	if v := d.Get("instance_name").(string); v != "" {
		args.InstanceName = v
	}

	if v := d.Get("description").(string); v != "" {
		args.Description = v
	}

	if v := d.Get("internet_charge_type").(string); v != "" {
		args.InternetChargeType = common.InternetChargeType(v)
	}

	if v := d.Get("internet_max_bandwidth_out").(int); v != 0 {
		args.InternetMaxBandwidthOut = v
	}

	if v := d.Get("host_name").(string); v != "" {
		args.HostName = v
	}

	if v := d.Get("password").(string); v != "" {
		args.Password = v
	}

	if v := d.Get("io_optimized").(string); v != "" {
		if v == "optimized" {
			args.IoOptimized = ecs.IoOptimized("true")
		} else {
			args.IoOptimized = ecs.IoOptimized("false")
		}
	}

	if v := d.Get("instance_charge_type").(string); v != "" {
		args.InstanceChargeType = common.InstanceChargeType(v)
	}

	log.Printf("[DEBUG] period is %d", d.Get("period").(int))
	if v := d.Get("period").(int); v != 0 {
		args.Period = v
	} else if args.InstanceChargeType == common.PrePaid {
		return nil, fmt.Errorf("period is required for instance_charge_type is PrePaid")
	}

	if v := d.Get("user_data").(string); v != "" {
		args.UserData = v
	}

	//In the VPC allocate public ip to ecs instance needs to set valid internet_max_bandwidth_out.
	subnetValue := d.Get("subnet_id").(string)
	vswitchValue := d.Get("vswitch_id").(string)
	if subnetValue != "" {
		if vswitchValue != "" && vswitchValue != subnetValue {
			return nil, fmt.Errorf("The value of 'subnet_id' is not equals 'vswitch_id', please unify or remove one of them.")
		}
		vswitchValue = subnetValue
	}

	if vswitchValue != string(NIL) {
		if d.Get("allocate_public_ip").(bool) && args.InternetMaxBandwidthOut <= 0 {
			if vswitchValue == "" {
				return nil, fmt.Errorf("Invalid internet_max_bandwidth_out result in allocation public ip failed in the default VPC.")
			}
			return nil, fmt.Errorf("Invalid internet_max_bandwidth_out result in allocation public ip failed in the specified VPC: %#v.", vswitchValue)
		}
		if vswitchValue != "" {
			args.VSwitchId = vswitchValue
		}
	}

	return args, nil
}

func userDataHashSum(user_data string) string {
	// Check whether the user_data is not Base64 encoded.
	// Always calculate hash of base64 decoded value since we
	// check against double-encoding when setting it
	v, base64DecodeError := base64.StdEncoding.DecodeString(user_data)
	if base64DecodeError != nil {
		v = []byte(user_data)
	}
	return string(v)
}
