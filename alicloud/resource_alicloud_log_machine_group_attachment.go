package alicloud

import (
	"fmt"
	"time"

	"strings"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAlicloudLogMachineGroupAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudLogMachineGroupAttachmentCreate,
		Read:   resourceAlicloudLogMachineGroupAttachmentRead,
		Update: resourceAlicloudLogMachineGroupAttachmentUpdate,
		Delete: resourceAlicloudLogMachineGroupAttachmentDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"config_names": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				MinItems: 1,
			},
		},
	}
}

func resourceAlicloudLogMachineGroupAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	project := d.Get("project").(string)
	group := d.Get("group_name").(string)

	if err := meta.(*AliyunClient).ApplyLogConfigToMachineGroup(project, group, d.Get("config_names").(*schema.Set).List()); err != nil {
		return err
	}

	d.SetId(fmt.Sprintf("%s%s%s", project, COLON_SEPARATED, group))

	return resourceAlicloudLogMachineGroupAttachmentRead(d, meta)
}

func resourceAlicloudLogMachineGroupAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	split := strings.Split(d.Id(), COLON_SEPARATED)

	configs, err := meta.(*AliyunClient).GetAppliedConfigs(split[0], split[1])
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("project", split[0])
	d.Set("group_name", split[1])
	d.Set("config_names", configs)

	return nil
}

func resourceAlicloudLogMachineGroupAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	split := strings.Split(d.Id(), COLON_SEPARATED)
	d.Partial(true)

	client := meta.(*AliyunClient)
	if d.HasChange("config_names") {
		old, new := d.GetChange("config_names")
		if err := client.RemoveLogConfigFromMachineGroup(split[0], split[1], expandStringList(old.(*schema.Set).List())); err != nil {
			return err
		}

		if err := client.ApplyLogConfigToMachineGroup(split[0], split[1], new.(*schema.Set).List()); err != nil {
			return err
		}

		d.SetPartial("config_names")
	}

	d.Partial(false)

	return resourceAlicloudLogMachineGroupAttachmentRead(d, meta)
}

func resourceAlicloudLogMachineGroupAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	split := strings.Split(d.Id(), COLON_SEPARATED)

	return resource.Retry(3*time.Minute, func() *resource.RetryError {
		configs, err := client.GetAppliedConfigs(split[0], split[1])
		if err != nil {
			if NotFoundError(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		if err := client.RemoveLogConfigFromMachineGroup(split[0], split[1], configs); err != nil {
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("Removing configs from machine group %s timeout.", split[1]))
	})
}
