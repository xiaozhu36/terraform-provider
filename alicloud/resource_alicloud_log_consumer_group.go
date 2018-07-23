package alicloud

import (
	"fmt"
	"time"

	"strings"

	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAlicloudLogConsumerGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudLogConsumerGroupCreate,
		Read:   resourceAlicloudLogConsumerGroupRead,
		Update: resourceAlicloudLogConsumerGroupUpdate,
		Delete: resourceAlicloudLogConsumerGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"logstore": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"in_order": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
		},
	}
}

func resourceAlicloudLogConsumerGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	project := d.Get("project").(string)
	logstore := d.Get("logstore").(string)
	groupName := d.Get("name").(string)

	if err := client.logconn.CreateConsumerGroup(project, logstore, sls.ConsumerGroup{
		ConsumerGroupName: groupName,
		Timeout:           d.Get("timeout").(int),
		InOrder:           d.Get("in_order").(bool),
	}); err != nil {
		return fmt.Errorf("CreateConsumerGroup got an error: %#v.", err)
	}

	d.SetId(fmt.Sprintf("%s%s%s%s%s", project, COLON_SEPARATED, logstore, COLON_SEPARATED, groupName))

	return resourceAlicloudLogConsumerGroupRead(d, meta)
}

func resourceAlicloudLogConsumerGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	split := strings.Split(d.Id(), COLON_SEPARATED)

	group, err := client.DescribeLogConsumerGroup(split[0], split[1], split[2])
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("DescribeLogConsumerGroup got an error: %#v.", err)
	}

	d.Set("project", split[0])
	d.Set("logstore", split[1])
	d.Set("name", group.ConsumerGroupName)
	d.Set("timeout", group.Timeout)
	d.Set("in_order", group.InOrder)

	return nil
}

func resourceAlicloudLogConsumerGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).logconn

	split := strings.Split(d.Id(), COLON_SEPARATED)
	d.Partial(true)

	update := false
	if d.HasChange("timeout") {
		update = true
		d.SetPartial("timeout")
	}
	if d.HasChange("in_order") {
		update = true
		d.SetPartial("in_order")
	}

	if update {
		if err := conn.UpdateConsumerGroup(split[0], split[1], sls.ConsumerGroup{
			ConsumerGroupName: split[2],
			Timeout:           d.Get("timeout").(int),
			InOrder:           d.Get("in_order").(bool),
		}); err != nil {
			return fmt.Errorf("UpdateLogConsumerGroup %s got an error: %#v.", split[2], err)
		}
	}
	d.Partial(false)

	return resourceAlicloudLogConsumerGroupRead(d, meta)
}

func resourceAlicloudLogConsumerGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	split := strings.Split(d.Id(), COLON_SEPARATED)

	return resource.Retry(3*time.Minute, func() *resource.RetryError {
		if err := client.logconn.DeleteConsumerGroup(split[0], split[1], split[2]); err != nil {
			return resource.NonRetryableError(fmt.Errorf("Deleting log machine group %s got an error: %#v", split[2], err))
		}

		if _, err := client.DescribeLogConsumerGroup(split[0], split[1], split[2]); err != nil {
			if NotFoundError(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("Deleting log machine group %s timeout.", split[2]))
	})
}
