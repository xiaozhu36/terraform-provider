package alicloud

import (
	"fmt"
	"time"

	"github.com/aliyun/aliyun-log-go-sdk"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func (client *AliyunClient) DescribeLogProject(name string) (project *sls.LogProject, err error) {
	project, err = client.logconn.GetProject(name)
	if err != nil {
		return project, fmt.Errorf("GetProject %s got an error: %#v.", name, err)
	}
	if project == nil || project.Name == "" {
		return project, GetNotFoundErrorFromString(GetNotFoundMessage("Log Project", name))
	}
	return
}

func (client *AliyunClient) DescribeLogStore(projectName, name string) (store *sls.LogStore, err error) {
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		store, err = client.logconn.GetLogStore(projectName, name)
		if err != nil {
			if IsExceptedErrors(err, []string{ProjectNotExist, LogStoreNotExist}) {
				return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Log Store", name)))
			}
			if IsExceptedErrors(err, []string{InternalServerError}) {
				return resource.RetryableError(fmt.Errorf("GetLogStore %s got an error: %#v.", name, err))
			}
			return resource.NonRetryableError(fmt.Errorf("GetLogStore %s got an error: %#v.", name, err))
		}
		return nil
	})

	if err != nil {
		return
	}

	if store == nil || store.Name == "" {
		return store, GetNotFoundErrorFromString(GetNotFoundMessage("Log Store", name))
	}
	return
}

func (client *AliyunClient) DescribeLogStoreIndex(projectName, name string) (index *sls.Index, err error) {
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		i, err := client.logconn.GetIndex(projectName, name)
		if err != nil {
			if IsExceptedErrors(err, []string{ProjectNotExist, LogStoreNotExist, IndexConfigNotExist}) {
				return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Log Store", name)))
			}
			if IsExceptedErrors(err, []string{InternalServerError}) {
				return resource.RetryableError(fmt.Errorf("GetLogStore %s got an error: %#v.", name, err))
			}
			return resource.NonRetryableError(fmt.Errorf("GetLogStore %s got an error: %#v.", name, err))
		}
		index = i
		return nil
	})

	if err != nil {
		return
	}

	if index == nil || index.Line == nil || len(index.Keys) < 1 {
		return index, GetNotFoundErrorFromString(GetNotFoundMessage("Log Store Index", name))
	}
	return
}

func (client *AliyunClient) DescribeLogConfig(projectName, name string) (config *sls.LogConfig, err error) {

	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		config, err = client.logconn.GetConfig(projectName, name)
		if err != nil {
			if IsExceptedErrors(err, []string{ConfigNotExist}) {
				return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Log Config", name)))
			}
			if IsExceptedErrors(err, []string{InternalServerError}) {
				return resource.RetryableError(fmt.Errorf("GetLogConfig %s got an error: %#v.", name, err))
			}
			return resource.NonRetryableError(fmt.Errorf("GetLogConfig %s got an error: %#v.", name, err))
		}
		return nil
	})

	if err != nil {
		return
	}

	if config == nil || config.Name == "" {
		return config, GetNotFoundErrorFromString(GetNotFoundMessage("Log Config", name))
	}
	return
}

func (client *AliyunClient) DescribeLogMachineGroup(projectName, groupName string) (group *sls.MachineGroup, err error) {

	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		group, err = client.logconn.GetMachineGroup(projectName, groupName)
		if err != nil {
			if IsExceptedErrors(err, []string{ProjectNotExist, GroupNotExist, MachineGroupNotExist}) {
				return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Log Machine Group", groupName)))
			}
			if IsExceptedErrors(err, []string{InternalServerError}) {
				return resource.RetryableError(fmt.Errorf("GetLogMachineGroup %s got an error: %#v.", groupName, err))
			}
			return resource.NonRetryableError(fmt.Errorf("GetLogMachineGroup %s got an error: %#v.", groupName, err))
		}
		return nil
	})

	if err != nil {
		return
	}

	if group == nil || group.Name == "" {
		return group, GetNotFoundErrorFromString(GetNotFoundMessage("Log Machine Group", groupName))
	}
	return
}

func (client *AliyunClient) DescribeLogConsumerGroup(project, logstore, groupName string) (group sls.ConsumerGroup, err error) {
	err = resource.Retry(2*time.Minute, func() *resource.RetryError {
		groups, err := client.logconn.ListConsumerGroup(project, logstore)
		if err != nil {
			if IsExceptedErrors(err, []string{ProjectNotExist, LogStoreNotExist}) {
				return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Consumer Group", groupName)))

			}
			if IsExceptedErrors(err, []string{InternalServerError}) {
				return resource.RetryableError(fmt.Errorf("GetLogConsumerGroup %s got an error: %#v.", groupName, err))
			}
			return resource.NonRetryableError(err)
		}
		if groups == nil || len(groups) < 1 {
			return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Consumer Group", groupName)))
		}

		for _, g := range groups {
			if g.ConsumerGroupName == groupName {
				group = *g
				return nil
			}
		}
		return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Consumer Group", groupName)))
	})
	return
}

func (client *AliyunClient) ApplyLogConfigToMachineGroup(project, groupName string, configs interface{}) error {
	configList := configs.(*schema.Set).List()
	if len(configList) < 1 {
		return nil
	}
	for _, c := range configList {
		if err := resource.Retry(2*time.Minute, func() *resource.RetryError {
			if err := client.logconn.ApplyConfigToMachineGroup(project, c.(string), groupName); err != nil {
				if IsExceptedErrors(err, []string{InternalServerError}) {
					return resource.RetryableError(err)
				}
				return resource.NonRetryableError(err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("Applying config %s to machine group %s got an error: %#v.", c.(string), groupName, err)
		}
	}
	return nil
}

func (client *AliyunClient) RemoveLogConfigFromMachineGroup(project, groupName string, configs []string) error {
	if len(configs) < 1 {
		return nil
	}
	for _, c := range configs {
		if err := resource.Retry(2*time.Minute, func() *resource.RetryError {
			if err := client.logconn.RemoveConfigFromMachineGroup(project, c, groupName); err != nil {
				if IsExceptedErrors(err, []string{InternalServerError}) {
					return resource.RetryableError(err)
				}
				if IsExceptedErrors(err, []string{GroupNotExist, ConfigNotExist}) {
					return nil
				}
				return resource.NonRetryableError(err)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("Removing config %s from machine group %s got an error: %#v.", c, groupName, err)
		}
	}
	return nil
}

func (client *AliyunClient) GetAppliedConfigs(project, groupName string) (configs []string, err error) {

	err = resource.Retry(3*time.Minute, func() *resource.RetryError {
		cfs, err := client.logconn.GetAppliedConfigs(project, groupName)
		if err != nil {
			if IsExceptedErrors(err, []string{ProjectNotExist, GroupNotExist}) {
				return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Machine Group configs", groupName)))
			}
			if IsExceptedErrors(err, []string{InternalServerError}) {
				return resource.RetryableError(fmt.Errorf("GetAppliedConfigs got an error: %#v.", err))
			}
			return resource.NonRetryableError(err)
		}
		if len(cfs) < 1 {
			return resource.NonRetryableError(GetNotFoundErrorFromString(GetNotFoundMessage("Machine Group configs", groupName)))
		}
		configs = cfs

		return nil
	})
	return
}
