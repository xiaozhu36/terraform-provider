package alicloud

import (
	"fmt"
	"testing"

	"strings"

	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAlicloudLogMachineGroupAttachment_basic(t *testing.T) {
	var project sls.LogProject
	var config1 sls.LogConfig
	var config2 sls.LogConfig
	var group sls.MachineGroup

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckAlicloudLogMachineGroupAttachmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccLogMachineGroupAttachment_basic(testInputDetail1, testInputDetail2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAlicloudLogProjectExists("alicloud_log_project.foo", &project),
					testAccCheckAlicloudLogConfigExists("alicloud_log_config.foo", &config1),
					testAccCheckAlicloudLogConfigExists("alicloud_log_config.bar", &config2),
					testAccCheckAlicloudLogMachineGroupExists("alicloud_log_machine_group.foo", &group),
					testAccCheckAlicloudLogMachineGroupAttachmentExists("alicloud_log_machine_group_attachment.foo", &group),
					resource.TestCheckResourceAttr("alicloud_log_machine_group_attachment.foo", "group_name", "ip"),
					resource.TestCheckResourceAttr("alicloud_log_machine_group_attachment.foo", "config_names.#", "2"),
				),
			},
		},
	})
}

func testAccCheckAlicloudLogMachineGroupAttachmentExists(name string, group *sls.MachineGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("Not found: %s", name)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Log machine group attachment ID is set")
		}

		split := strings.Split(rs.Primary.ID, COLON_SEPARATED)

		if _, err := testAccProvider.Meta().(*AliyunClient).GetAppliedConfigs(split[0], split[1]); err != nil {
			return err
		}
		return nil
	}
}

func testAccCheckAlicloudLogMachineGroupAttachmentDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*AliyunClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "alicloud_log_machine_group_attachment" {
			continue
		}

		split := strings.Split(rs.Primary.ID, COLON_SEPARATED)

		if _, err := client.GetAppliedConfigs(split[0], split[1]); err != nil {
			if NotFoundError(err) {
				return nil
			}
			return fmt.Errorf("Check log machine group configs got an error: %#v.", err)
		}

		return fmt.Errorf("Log machine group %s configs still exist.", rs.Primary.ID)
	}

	return nil
}

func testAccLogMachineGroupAttachment_basic(input1, input2 string) string {
	return fmt.Sprintf(`
variable "name" {
    default = "tf-test-log-machine-group-attachment"
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
resource "alicloud_log_config" "bar" {
    project = "${alicloud_log_project.foo.name}"
    logstore = "${alicloud_log_store.foo.name}"
    name = "${var.name}-2"
    input_detail = <<DEFINITION
    %s
    DEFINITION
}
resource "alicloud_log_machine_group" "foo" {
    project = "${alicloud_log_project.foo.name}"
    name = "${var.name}"
    identify_type = "userdefined"
    topic = "terraform"
    identify_list = ["terraform", "abc1234"]
}
resource "alicloud_log_machine_group_attachment" "foo" {
    project = "${alicloud_log_project.foo.name}"
    group_name = "${alicloud_log_machine_group.foo.name}"
    config_names = ["${alicloud_log_config.foo.name}", "${alicloud_log_config.bar.name}"]
}
`, input1, input2)
}

var testInputDetail1 = `
slave-java:
  image: 'registry.aliyuncs.com/acs-sample/jenkins-slave-dind-java'
  volumes:
      - /var/run/docker.sock:/var/run/docker.sock
  restart: always
  labels:
      aliyun.scale: '1'
`

var testInputDetail2 = `
slave-java:
  image: 'registry.aliyuncs.com/acs-sample/jenkins-slave-dind-java'
  volumes:
      - /var/run/docker.sock:/var/run/docker.sock
  restart: always
  labels:
      aliyun.scale: '1'
`
