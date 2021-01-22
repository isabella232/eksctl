package iam

import (
	"fmt"

	"github.com/weaveworks/eksctl/pkg/cfn/manager"

	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
	"github.com/weaveworks/eksctl/pkg/utils/tasks"

	"github.com/kris-nova/logger"
	"github.com/weaveworks/eksctl/pkg/ctl/cmdutils"
	"github.com/weaveworks/eksctl/pkg/kubernetes"
)

func (m *Manager) Delete(serviceAccounts []string, plan, wait bool) error {
	taskTree, err := m.NewTasksToDeleteIAMServiceAccounts(serviceAccounts, kubernetes.NewCachedClientSet(m.clientSet), wait)
	if err != nil {
		return err
	}
	taskTree.PlanMode = plan

	logger.Info(taskTree.Describe())
	if errs := taskTree.DoAllSync(); len(errs) > 0 {
		logger.Info("%d error(s) occurred and IAM Role stacks haven't been deleted properly, you may wish to check CloudFormation console", len(errs))
		for _, err := range errs {
			logger.Critical("%s\n", err.Error())
		}
		return fmt.Errorf("failed to delete iamserviceaccount(s)")
	}

	cmdutils.LogPlanModeWarning(plan && len(serviceAccounts) > 0)
	return nil
}

// NewTasksToDeleteIAMServiceAccounts defines tasks required to delete all of the iamserviceaccounts
func (m *Manager) NewTasksToDeleteIAMServiceAccounts(serviceAccounts []string, clientSetGetter kubernetes.ClientSetGetter, wait bool) (*tasks.TaskTree, error) {
	serviceAccountStacks, err := m.stackManager.DescribeIAMServiceAccountStacks()
	if err != nil {
		return nil, err
	}

	taskTree := &tasks.TaskTree{Parallel: true}

	serviceAccountToDeleteMap := listStringToSet(serviceAccounts)

	for _, s := range serviceAccountStacks {
		saTasks := &tasks.TaskTree{
			Parallel:  false,
			IsSubTask: true,
		}
		name := manager.GetIAMServiceAccountName(s)

		if _, shouldDelete := serviceAccountToDeleteMap[name]; !shouldDelete {
			continue
		}

		info := fmt.Sprintf("delete IAM role for serviceaccount %q", name)
		if wait {
			saTasks.Append(manager.NewTaskWithStackSpec(info, s, m.stackManager.DeleteStackBySpecSync))
		} else {
			saTasks.Append(manager.NewAsyncTaskWithStackSpec(info, s, m.stackManager.DeleteStackBySpec))
		}
		call := func(clientSet kubernetes.Interface) error {
			meta, err := api.ClusterIAMServiceAccountNameStringToClusterIAMMeta(name)
			if err != nil {
				return err
			}
			return kubernetes.MaybeDeleteServiceAccount(clientSet, meta.AsObjectMeta())
		}
		saTasks.Append(manager.NewKubernetesTask(fmt.Sprintf("delete serviceaccount %q", name), clientSetGetter, call))
		taskTree.Append(saTasks)
	}

	return taskTree, nil
}

func listStringToSet(values []string) map[string]struct{} {
	stacksMap := make(map[string]struct{})
	for _, value := range values {
		stacksMap[value] = struct{}{}
	}
	return stacksMap
}
