package auth

type Operation string

const (
	OpAsk                   Operation = "ask"
	OpSearch                Operation = "search"
	OpBrowse                Operation = "browse"
	OpDocumentUpload        Operation = "document.upload"
	OpFactDecision          Operation = "fact.decision"
	OpEntityMerge           Operation = "entity.merge"
	OpContradictionDecision Operation = "contradiction.decision"
)

var operationRoles = map[Operation][]string{
	OpAsk:                   {RoleResearcher, RoleAnalyst, RoleManager, RoleExpert, RoleAdmin, RolePartner},
	OpSearch:                {RoleResearcher, RoleAnalyst, RoleManager, RoleExpert, RoleAdmin},
	OpBrowse:                {RoleResearcher, RoleAnalyst, RoleManager, RoleExpert, RoleAdmin},
	OpDocumentUpload:        {RoleAnalyst, RoleAdmin},
	OpFactDecision:          {RoleExpert, RoleAdmin},
	OpEntityMerge:           {RoleExpert, RoleAdmin},
	OpContradictionDecision: {RoleExpert, RoleAdmin},
}

func Allowed(operation Operation, principal Principal) bool {
	roles, ok := operationRoles[operation]
	if !ok {
		return false
	}
	return principal.HasAnyRole(roles...)
}
