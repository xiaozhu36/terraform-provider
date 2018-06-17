package alicloud

import (
	"fmt"
	"testing"

	"strings"

	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAlicloudLogConfig_basic(t *testing.T) {
	var project sls.LogProject
	var store sls.LogStore
	var config sls.LogConfig

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAlicloudLogConfigDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLogConfig_basic(testInputDetail1),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudLogProjectExists("alicloud_log_project.foo", &project),
					testAccCheckAlicloudLogStoreExists("alicloud_log_store.foo", &store),
					testAccCheckAlicloudLogConfigExists("alicloud_log_config.foo", &config),
					resource.TestCheckResourceAttr("alicloud_log_config.foo", "name", "tf-test-log-config"),
				),
			},
		},
	})
}

func testAccCheckAlicloudLogConfigExists(name string, config *sls.LogConfig) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Log machine group attachment ID is set")
		}

		split := strings.Split(rs.Primary.ID, COLON_SEPARATED)

		c, err := testAccProvider.Meta().(*AliyunClient).DescribeLogConfig(split[0], split[1])
		if err != nil {
			return err
		}
		config = c
		return nil
	}
}

func testAccCheckAlicloudLogConfigDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_log_config" {
			continue
		}

		split := strings.Split(rs.Primary.ID, COLON_SEPARATED)

		config, err := client.DescribeLogConfig(split[0], split[1])
		if err != nil {
			if NotFoundError(err) {
				return nil
			}
			return fmt.Errorf("Check log config got an error: %#v.", err)
		}

		if config.Name != "" {
			return fmt.Errorf("Log config %s still exist.", rs.Primary.ID)
		}
	}

	return nil
}

func testAccLogConfig_basic(input string) string {
	return fmt.Sprintf(`
variable "name" {
    default = "tf-test-log-config"
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

resource "alicloud_log_config" "foo" {
    project = "${alicloud_log_project.foo.name}"
    logstore = "${alicloud_log_store.foo.name}"
    name = "${var.name}-1"
    input_detail = <<DEFINITION
    %s
    DEFINITION
}
`, input)
}
