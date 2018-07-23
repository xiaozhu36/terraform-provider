package alicloud

import (
	"fmt"
	"time"

	"github.com/aliyun/fc-go-sdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAlicloudFCService() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudFCServiceCreate,
		Read:   resourceAlicloudFCServiceRead,
		Update: resourceAlicloudFCServiceUpdate,
		Delete: resourceAlicloudFCServiceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateStringLengthInRange(1, 128),
			},
			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// uuid is 26 characters, limit the prefix to 229.
					value := v.(string)
					if len(value) > 122 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 102 characters, name is limited to 128", k))
					}
					return
				},
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"internet_access": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"role": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"logstore": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"vswitch_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				MinItems: 1,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"last_modified": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAlicloudFCServiceCreate(d *schema.ResourceData, meta interface{}) error {
	if err := requireAccountId(meta); err != nil {
		return err
	}
	client := meta.(*AliyunClient)
	conn := client.fcconn

	var name string
	if v, ok := d.GetOk("name"); ok {
		name = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		name = resource.PrefixedUniqueId(v.(string))
	} else {
		name = resource.UniqueId()
	}

	if err := ensureLogstoreExist(d, meta); err != nil {
		return err
	}
	input := &fc.CreateServiceInput{
		ServiceName:    StringPointer(name),
		Description:    StringPointer(d.Get("description").(string)),
		InternetAccess: BoolPointer(d.Get("internet_access").(bool)),
		Role:           StringPointer(d.Get("role").(string)),
		LogConfig: &fc.LogConfig{
			Project:  StringPointer(d.Get("project").(string)),
			Logstore: StringPointer(d.Get("logstore").(string)),
		},
	}
	vpcconfig, err := getVpcConfig(d, meta)
	if err != nil {
		return err
	}
	input.VPCConfig = vpcconfig

	var service *fc.CreateServiceOutput
	if err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		service, err = conn.CreateService(input)
		if err != nil {
			if IsExceptedErrors(err, []string{AccessDenied, "does not exist"}) {
				return resource.RetryableError(fmt.Errorf("Error creating function compute service got an error: %#v", err))
			}
			return resource.NonRetryableError(fmt.Errorf("Error creating function compute service got an error: %#v", err))
		}
		return nil

	}); err != nil {
		return err
	}

	if service == nil {
		return fmt.Errorf("Creating function compute service got a empty response: %#v.", service)
	}

	d.SetId(*service.ServiceName)

	return resourceAlicloudFCServiceRead(d, meta)
}

func resourceAlicloudFCServiceRead(d *schema.ResourceData, meta interface{}) error {
	if err := requireAccountId(meta); err != nil {
		return err
	}

	client := meta.(*AliyunClient)

	service, err := client.DescribeFcService(d.Id())
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("DescribeFCService %s got an error: %#v", d.Id(), err)
	}

	d.Set("name", service.ServiceName)
	d.Set("description", service.Description)
	d.Set("internet_access", service.InternetAccess)
	d.Set("role", service.Role)
	if logconfig := service.LogConfig; logconfig != nil {
		d.Set("project", logconfig.Project)
		d.Set("logstore", logconfig.Logstore)
	}
	if vpcConfig := service.VPCConfig; vpcConfig != nil {
		d.Set("security_group_id", vpcConfig.SecurityGroupID)
		d.Set("vswitch_ids", vpcConfig.VSwitchIDs)
		d.Set("vpc_id", vpcConfig.VPCID)
	}
	d.Set("last_modified", service.LastModifiedTime)

	return nil
}

func resourceAlicloudFCServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	if err := requireAccountId(meta); err != nil {
		return err
	}
	client := meta.(*AliyunClient)

	d.Partial(true)
	updateInput := &fc.UpdateServiceInput{}

	if d.HasChange("role") {
		updateInput.Role = StringPointer(d.Get("role").(string))
		d.SetPartial("role")
	}
	if d.HasChange("internet_access") {
		updateInput.InternetAccess = BoolPointer(d.Get("internet_access").(bool))
		d.SetPartial("internet_access")
	}
	if d.HasChange("description") {
		updateInput.Description = StringPointer(d.Get("description").(string))
		d.SetPartial("description")
	}
	if d.HasChange("project") {
		updateInput.LogConfig.Project = StringPointer(d.Get("project").(string))
		d.SetPartial("project")
	}
	if d.HasChange("logstore") {
		updateInput.LogConfig.Logstore = StringPointer(d.Get("logstore").(string))
		d.SetPartial("logstore")
	}
	if d.HasChange("vswitch_ids") || d.HasChange("security_group_id") {
		vpcconfig, err := getVpcConfig(d, meta)
		if err != nil {
			return err
		}
		updateInput.VPCConfig = vpcconfig
		d.SetPartial("vswitch_ids")
		d.SetPartial("security_group_id")
	}

	if updateInput != nil {
		updateInput.ServiceName = StringPointer(d.Id())
		if _, err := client.fcconn.UpdateService(updateInput); err != nil {
			return fmt.Errorf("UpdateService %s got an error: %#v.", d.Id(), err)
		}
	}

	d.Partial(false)
	return resourceAlicloudFCServiceRead(d, meta)
}

func resourceAlicloudFCServiceDelete(d *schema.ResourceData, meta interface{}) error {
	if err := requireAccountId(meta); err != nil {
		return err
	}
	client := meta.(*AliyunClient)

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		if _, err := client.fcconn.DeleteService(&fc.DeleteServiceInput{
			ServiceName: StringPointer(d.Id()),
		}); err != nil {
			if IsExceptedErrors(err, []string{ServiceNotFound}) {
				return nil
			}
			return resource.NonRetryableError(fmt.Errorf("Deleting function service got an error: %#v.", err))
		}

		if _, err := client.DescribeFcService(d.Id()); err != nil {
			if NotFoundError(err) {
				return nil
			}
			return resource.RetryableError(fmt.Errorf("While deleting service, getting service %s got an error: %#v.", d.Id(), err))
		}
		return nil
	})

}

func getVpcConfig(d *schema.ResourceData, meta interface{}) (config *fc.VPCConfig, err error) {
	vswitch_ids := d.Get("vswitch_ids").(*schema.Set).List()
	if len(vswitch_ids) > 0 {
		sg, ok := d.GetOk("security_group_id")
		if !ok || sg == "" {
			err = fmt.Errorf("'security_group_id' is required when 'vswitch_ids' is set.")
			return
		}
		if role, ok := d.GetOk("role"); !ok || role.(string) == "" {
			err = fmt.Errorf("'role' is required when 'vswitch_ids' is set.")
			return
		}
		vsw, e := meta.(*AliyunClient).DescribeVswitch(vswitch_ids[0].(string))
		if e != nil {
			err = fmt.Errorf("While creating fc service, describing vswitch %s got an error: %#v.", vswitch_ids[0].(string), e)
			return
		}
		config = &fc.VPCConfig{
			VSwitchIDs:      expandStringList(d.Get("vswitch_ids").(*schema.Set).List()),
			SecurityGroupID: StringPointer(sg.(string)),
			VPCID:           StringPointer(vsw.VpcId),
		}
	}
	return
}

func ensureLogstoreExist(d *schema.ResourceData, meta interface{}) (err error) {
	project := d.Get("project").(string)
	if project != "" {
		err = resource.Retry(2*time.Minute, func() *resource.RetryError {
			if _, e := meta.(*AliyunClient).logconn.CheckProjectExist(project); e != nil {
				if NotFoundError(e) {
					return resource.RetryableError(fmt.Errorf("Check log project %s failed: %#v.", project, e))
				}
				return resource.NonRetryableError(fmt.Errorf("Check log project %s failed: %#v.", project, e))
			}
			return nil
		})
	}

	if err != nil {
		return
	}

	if logstore := d.Get("logstore").(string); logstore != "" {
		err = resource.Retry(2*time.Minute, func() *resource.RetryError {
			if _, e := meta.(*AliyunClient).logconn.CheckLogstoreExist(project, logstore); e != nil {
				if NotFoundError(e) {
					return resource.RetryableError(fmt.Errorf("Check logstore %s failed: %#v.", logstore, e))
				}
				return resource.NonRetryableError(fmt.Errorf("Check logstore %s failed: %#v.", logstore, e))
			}
			return nil
		})
	}
	return
}
