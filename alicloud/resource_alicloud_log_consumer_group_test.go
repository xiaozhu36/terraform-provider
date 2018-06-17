package alicloud

import (
	"fmt"
	"testing"

	"strings"

	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAlicloudLogConsumerGroup_basic(t *testing.T) {
	var project sls.LogProject
	var store sls.LogStore
	var group sls.ConsumerGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAlicloudLogConsumerGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAlicloudLogConsumerGroupBasic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudLogProjectExists("alicloud_log_project.foo", &project),
					testAccCheckAlicloudLogStoreExists("alicloud_log_store.foo", &store),
					testAccCheckAlicloudLogConsumerGroupExists("alicloud_log_consumer_group.foo", &group),
					resource.TestCheckResourceAttr("alicloud_log_consumer_group.foo", "name", "tf-test-log-consumer_group"),
					resource.TestCheckResourceAttr("alicloud_log_consumer_group.foo", "timeout", "100"),
					resource.TestCheckResourceAttr("alicloud_log_consumer_group.foo", "name", "true"),
				),
			},
		},
	})
}

func testAccCheckAlicloudLogConsumerGroupExists(name string, group *sls.ConsumerGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Log machine group attachment ID is set")
		}

		split := strings.Split(rs.Primary.ID, COLON_SEPARATED)

		g, err := testAccProvider.Meta().(*AliyunClient).DescribeLogConsumerGroup(split[0], split[1], split[2])
		if err != nil {
			return err
		}
		group = &g
		return nil
	}
}

func testAccCheckAlicloudLogConsumerGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_log_consumer_group" {
			continue
		}

		split := strings.Split(rs.Primary.ID, COLON_SEPARATED)

		group, err := client.DescribeLogConsumerGroup(split[0], split[1], split[2])
		if err != nil {
			if NotFoundError(err) {
				return nil
			}
			return fmt.Errorf("Check log config got an error: %#v.", err)
		}

		if group.ConsumerGroupName != "" {
			return fmt.Errorf("Log consumer group %s still exist.", rs.Primary.ID)
		}
	}

	return nil
}

const testAlicloudLogConsumerGroupBasic = `
variable "name" {
    default = "tf-test-log-consumer_group"
}
resource "alicloud_log_project" "foo" {
    name = "${var.name}"
    description = "tf unit test"
}

resource "alicloud_log_store" "foo" {
    project = "${alicloud_log_project.foo.name}"
    name = "${var.name}"
    retention_period = "3000"
    shard_count = 1
}

resource "alicloud_log_consumer_group" "foo" {
    project = "${alicloud_log_project.foo.name}"
    logstore = "${alicloud_log_store.foo.name}"
    name = "${var.name}"
    timeout = "100"
    in_order = "true"
}
`
