package iam

import (
	api "github.com/weaveworks/eksctl/pkg/apis/eksctl.io/v1alpha5"
	"github.com/weaveworks/eksctl/pkg/ctl/cmdutils"
	"github.com/weaveworks/eksctl/pkg/kubernetes"
)

func (a *Manager) CreateIAMServiceAccount(iamServiceAccounts []*api.ClusterIAMServiceAccount, plan bool) error {
	taskTree := a.stackManager.NewTasksToCreateIAMServiceAccounts(iamServiceAccounts, a.oidcManager, kubernetes.NewCachedClientSet(a.clientSet), false)
	taskTree.PlanMode = plan

	err := doTasks(taskTree)

	cmdutils.LogPlanModeWarning(plan && len(iamServiceAccounts) > 0)

	return err
}
