//provider "alicloud" {
//  account_id = "${var.account}"
//  region = "${var.region}"
//}
//variable "region" {
//  default = "cn-hongkong"
//}
//variable "account" {
//  default = "12345"
//}
//variable "name" {
//  default = "tf-fc-example"
//}
//
//resource "alicloud_log_project" "foo" {
//  name = "${var.name}"
//  description = "tf unit test"
//}
//resource "alicloud_log_store" "bar" {
//  project = "${alicloud_log_project.foo.name}"
//  name = "${var.name}-source"
//  retention_period = "3000"
//  shard_count = 1
//}
//resource "alicloud_log_store" "foo" {
//  project = "${alicloud_log_project.foo.name}"
//  name = "${var.name}"
//  retention_period = "3000"
//  shard_count = 1
//}
//
//resource "alicloud_fc_service" "foo" {
//  name = "${var.name}"
////  vswitch_ids = ["${alicloud_vswitch.vsw.id}"]
////  security_group_id = "${alicloud_security_group.group.id}"
//  internet_access = false
////  role = "${alicloud_ram_role.role.arn}"
////  depends_on = ["alicloud_ram_role_policy_attachment.attac", "alicloud_ram_role_policy_attachment.attac2"]
//}
//
//resource "alicloud_fc_function" "foo" {
//  service = "${alicloud_fc_service.foo.name}"
//  name = "${var.name}"
//  filename = "hello.zip"
////  oss_bucket = "${alicloud_oss_bucket.foo.id}"
////  oss_key = "${alicloud_oss_bucket_object.foo.key}"
//  memory_size = 512
//  runtime = "python2.7"
//}
//
//resource "alicloud_fc_trigger" "foo" {
//  service = "${alicloud_fc_service.foo.name}"
//  function = "${alicloud_fc_function.foo.name}"
//  name = "${var.name}"
//  role = "${alicloud_ram_role.foo.arn}"
//  source_arn = "acs:log:cn-hongkong:${var.account}:project/${alicloud_log_project.foo.name}"
//  //  oss_bucket = "${alicloud_oss_bucket.foo.id}"
//  //  oss_key = "${alicloud_oss_bucket_object.foo.key}"
//  type = "log"
////  config = "${file("trigger-http.json")}"
//  config = <<EOF
//    {
//        "sourceConfig": {
//            "project": "${alicloud_log_project.foo.name}",
//            "logstore": "${alicloud_log_store.bar.name}"
//        },
//        "jobConfig": {
//            "maxRetryTime": 3,
//            "triggerInterval": 60
//        },
//        "functionParameter": {
//            "a": "b",
//            "c": "d"
//        },
//        "logConfig": {
//            "project": "${alicloud_log_project.foo.name}",
//            "logstore": "${alicloud_log_store.foo.name}"
//        },
//        "enable": true
//    }
//  EOF
//  depends_on = ["alicloud_ram_role_policy_attachment.foo"]
//}
//
//resource "alicloud_ram_role" "foo" {
//  name = "${var.name}-trigger"
//  document = <<EOF
//    {
//      "Statement": [
//        {
//          "Action": "sts:AssumeRole",
//          "Effect": "Allow",
//          "Principal": {
//            "Service": [
//              "log.aliyuncs.com"
//            ]
//          }
//        }
//      ],
//      "Version": "1"
//    }
//  EOF
//  description = "this is a test"
//  force = true
//}
//
//resource "alicloud_ram_policy" "foo" {
//  name = "${var.name}-trigger"
////  document = "${file("trigger_policy.json")}"
//  document = <<EOF
//    {
//      "Version": "1",
//      "Statement": [
//        {
//          "Action": [
//            "fc:InvokeFunction"
//          ],
//          "Resource": "acs:fc:*:*:services/${alicloud_fc_service.foo.name}/functions/*",
//          "Effect": "Allow"
//        },
//        {
//          "Action": [
//            "log:Get*",
//            "log:List*",
//            "log:PostLogStoreLogs",
//            "log:CreateConsumerGroup",
//            "log:UpdateConsumerGroup",
//            "log:DeleteConsumerGroup",
//            "log:ListConsumerGroup",
//            "log:ConsumerGroupUpdateCheckPoint",
//            "log:ConsumerGroupHeartBeat",
//            "log:GetConsumerGroupCheckPoint"
//          ],
//          "Resource": "*",
//          "Effect": "Allow"
//        }
//      ]
//    }
//  EOF
//  description = "this is a test"
//  force = true
//}
//resource "alicloud_ram_role_policy_attachment" "foo" {
//  role_name = "${alicloud_ram_role.foo.name}"
//  policy_name = "${alicloud_ram_policy.foo.name}"
//  policy_type = "Custom"
//}


//resource "alicloud_ram_role" "role" {
//  name = "fc-role"
//  document = "${file("role.json")}"
//  description = "this is a test"
//  force = true
//}
//
//resource "alicloud_ram_policy" "policy" {
//  name = "fc-policy"
//  document = "${file("logpolicy.json")}"
//  description = "this is a test"
//  force = true
//}
//
//resource "alicloud_ram_role_policy_attachment" "attac" {
//  role_name = "${alicloud_ram_role.role.name}"
//  policy_name = "${alicloud_ram_policy.policy.name}"
//  policy_type = "Custom"
//}
//
//resource "alicloud_ram_policy" "vpcpolicy" {
//  name = "fc-policy-for-vpc"
//    document = "${file("policy.json")}"
//  description = "this is a test"
//  force = true
//}
//resource "alicloud_ram_role_policy_attachment" "attac2" {
//  role_name = "${alicloud_ram_role.role.name}"
//  policy_name = "${alicloud_ram_policy.vpcpolicy.name}"
//  policy_type = "Custom"
//}
//resource "alicloud_oss_bucket" "foo" {
//  bucket = "${var.name}"
//}
//resource "alicloud_oss_bucket_object" "foo" {
//  bucket = "${alicloud_oss_bucket.foo.id}"
//  key = "fc/hello.zip"
//  source = "hello.zip"
//}



provider "alicloud" {
  account_id = "${var.account}"
  region = "${var.region}"
}
variable "region" {
  default = "cn-hongkong"
}
variable "account" {
  default = ""
}

variable "name" {
  default = "test-acc-alicloud-fc-function-basic"
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
resource "alicloud_fc_service" "foo" {
  name = "${var.name}"
  description = "tf unit test"
  project = "${alicloud_log_project.foo.name}"
  logstore = "${alicloud_log_store.foo.name}"
  role = "${alicloud_ram_role.foo.arn}"
  depends_on = ["alicloud_ram_role_policy_attachment.foo"]
}

resource "alicloud_oss_bucket" "foo" {
  bucket = "${var.name}"
}

resource "alicloud_oss_bucket_object" "foo" {
  bucket = "${alicloud_oss_bucket.foo.id}"
  key = "fc/hello.zip"
  content = <<EOF
  	# -*- coding: utf-8 -*-
	def handler(event, context):
	    print "hello world"
	    return 'hello world'
  EOF
}

resource "alicloud_fc_function" "foo" {
  service = "${alicloud_fc_service.foo.name}"
  name = "${var.name}"
  oss_bucket = "${alicloud_oss_bucket.foo.id}"
  oss_key = "${alicloud_oss_bucket_object.foo.key}"
  runtime = "nodejs6"
}
resource "alicloud_ram_role" "foo" {
  name = "${var.name}"
  document = <<EOF
  {
    "Statement": [
      {
        "Action": "sts:AssumeRole",
        "Effect": "Allow",
        "Principal": {
          "Service": [
            "fc.aliyuncs.com"
          ]
        }
      }
    ],
    "Version": "1"
  }
  EOF
  description = "this is a test"
  force = true
}

resource "alicloud_ram_role_policy_attachment" "foo" {
  role_name = "${alicloud_ram_role.foo.name}"
  policy_name = "AliyunLogFullAccess"
  policy_type = "System"
}