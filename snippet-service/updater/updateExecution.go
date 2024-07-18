package updater

import "github.com/plgd-dev/hub/v2/snippet-service/pb"

type executeByType int

const (
	executeByTypeFindCondition executeByType = iota
	executeByTypeCondition
	executeByTypeOnDemand
)

type appliedCondition struct {
	id      string
	version uint64
	token   string
}

type appliedOnDemand struct {
	token string
}

type execution struct {
	condition  appliedCondition // executedBy = executeByTypeCondition
	conditions []*pb.Condition  // executedBy = executeByTypeFindCondition
	onDemand   appliedOnDemand  // executedBy = executeByTypeOnDemand
	force      bool
	executeBy  executeByType
}

func (e *execution) token() string {
	if e.executeBy == executeByTypeOnDemand {
		return e.onDemand.token
	}
	if e.executeBy == executeByTypeCondition {
		return e.condition.token
	}
	return ""
}

func (e *execution) setCondition(c appliedCondition) {
	e.condition = c
	e.conditions = nil
	e.executeBy = executeByTypeCondition
}

func (e *execution) setExecutedBy(ac *pb.AppliedConfiguration) {
	if e.executeBy == executeByTypeOnDemand {
		ac.ExecutedBy = pb.MakeExecutedByOnDemand()
		return
	}
	if e.condition.id != "" {
		ac.ExecutedBy = pb.MakeExecutedByConditionId(e.condition.id, e.condition.version)
		return
	}
	firstCondition := e.conditions[0]
	ac.ExecutedBy = pb.MakeExecutedByConditionId(firstCondition.GetId(), firstCondition.GetVersion())
}

type executionResult struct {
	validUntil int64
	condition  appliedCondition // executedBy = executeByTypeCondition or executeByTypeFindCondition
	onDemand   appliedOnDemand  // executedBy = executeByTypeOnDemand
	executedBy executeByType
	err        error
}

func (er executionResult) token() string {
	if er.executedBy == executeByTypeOnDemand {
		return er.onDemand.token
	}
	return er.condition.token
}
