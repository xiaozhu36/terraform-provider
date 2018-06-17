//resource "alicloud_log_project" "pp" {
//  name = "terraform"
//}
//
//resource "alicloud_log_store" "store" {
//  project = "${alicloud_log_project.pp.id}"
//  name = "terraform2"
//  retention_period = "3600"
//  shard_count = 1
//}
//
//resource "alicloud_log_store_index" "index" {
//  project = "${alicloud_log_project.pp.id}"
//  logstore = "${alicloud_log_store.store.name}"
//  index_type = "Field"
//  field_name = "tef"
//}
//
//resource "alicloud_log_store_index" "index2" {
//  project = "${alicloud_log_project.pp.id}"
//  logstore = "${alicloud_log_store.store.name}"
//  index_type = "Field"
//  field_name = "teff"
//}
//
//resource "alicloud_log_store_index" "index3" {
//  project = "${alicloud_log_project.pp.id}"
//  logstore = "${alicloud_log_store.store.name}"
////  index_type = "Field"
////  field_name = "tef2"
//  token = " #$%^*\r\n\t"
//}
//
//resource "alicloud_log_config" "config" {
//  project = "${alicloud_log_project.pp.id}"
//  name = "terraform"
//  input_detail = "${file("log_config.json")}"
//}
//
//resource "alicloud_log_machine_group" "group" {
//  project = "${alicloud_log_project.pp.name}"
//  name = "from-terraform"
//  topic = "terraform"
//  identify_list = ["10.0.0.1", "10.0.0.3", "10.0.0.2"]
//}
//
//resource "alicloud_log_machine_group_attachment" "config" {
//  project = "${alicloud_log_project.pp.name}"
//  group_name = "${alicloud_log_machine_group.group.name}"
//  config_names = ["${alicloud_log_config.config.name}"]
//}
//
//resource "alicloud_log_consumer_group" "group" {
//  project = "${alicloud_log_project.pp.name}"
//  logstore = "${alicloud_log_store.store.name}"
//  name = "from-terraform"
//  timeout = "100"
//  in_order = "true"
//}

resource "alicloud_log_project" "aa" {
  name = "for-tf-test"
  description = "tf unit test"
}