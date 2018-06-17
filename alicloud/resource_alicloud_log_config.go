package alicloud

import (
	"fmt"
	"time"

	"strings"

	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAlicloudLogConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceAlicloudLogConfigCreate,
		Read:   resourceAlicloudLogConfigRead,
		Update: resourceAlicloudLogConfigUpdate,
		Delete: resourceAlicloudLogConfigDelete,
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
				Optional: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"input_detail": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAlicloudLogConfigCreate(d *schema.ResourceData, meta interface{}) error {

	project := d.Get("project").(string)
	if err := meta.(*AliyunClient).logconn.CreateConfig(project, &sls.LogConfig{
		Name:        d.Get("name").(string),
		InputType:   sls.InputTypeFile,
		InputDetail: d.Get("input_detail").(sls.InputDetailInterface),
		OutputType:  "LogService",
		OutputDetail: sls.OutputDetail{
			LogStoreName: d.Get("logstore").(string),
		},
	}); err != nil {
		return fmt.Errorf("CreateLogConfig got an error: %#v.", err)
	}

	d.SetId(fmt.Sprintf("%s%s%s", project, COLON_SEPARATED, d.Get("name").(string)))

	return resourceAlicloudLogConfigUpdate(d, meta)
}

func resourceAlicloudLogConfigRead(d *schema.ResourceData, meta interface{}) error {
	split := strings.Split(d.Id(), COLON_SEPARATED)

	config, err := meta.(*AliyunClient).DescribeLogConfig(split[0], split[1])
	if err != nil {
		if NotFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("GetConfig %s got an error: %#v.", split[1], err)
	}

	d.Set("project", split[0])
	d.Set("name", config.Name)
	d.Set("input_detail", config.InputDetail.(string))
	d.Set("logstore", config.OutputDetail.LogStoreName)

	return nil
}

func resourceAlicloudLogConfigUpdate(d *schema.ResourceData, meta interface{}) error {

	split := strings.Split(d.Id(), COLON_SEPARATED)
	d.Partial(true)

	update := false
	if d.HasChange("input_detail") {
		update = true
		d.SetPartial("input_detail")
	}
	if d.HasChange("logstore") {
		update = true
		d.SetPartial("logstore")
	}

	if update {
		if err := meta.(*AliyunClient).logconn.UpdateConfig(split[0], &sls.LogConfig{
			Name:        split[1],
			InputType:   "file",
			InputDetail: d.Get("input_detail").(string),
			OutputType:  "LogService",
			OutputDetail: sls.OutputDetail{
				LogStoreName: d.Get("logstore").(string),
			},
		}); err != nil {
			return fmt.Errorf("UpdateLogConfig %s got an error: %#v.", split[1], err)
		}
	}
	d.Partial(false)

	return resourceAlicloudLogConfigRead(d, meta)
}

func resourceAlicloudLogConfigDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	split := strings.Split(d.Id(), COLON_SEPARATED)

	return resource.Retry(3*time.Minute, func() *resource.RetryError {
		if err := client.logconn.DeleteConfig(split[0], split[1]); err != nil {
			return resource.NonRetryableError(fmt.Errorf("Deleting log store %s got an error: %#v", split[1], err))
		}

		if _, err := client.DescribeLogConfig(split[0], split[1]); err != nil {
			if NotFoundError(err) {
				d.SetId("")
				return nil
			}
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("Deleting log config %s timeout.", split[1]))
	})
}
