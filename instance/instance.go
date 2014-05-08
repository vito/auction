package instance

import "github.com/onsi/auction/util"

type Instance struct {
	AppGuid           string
	InstanceGuid      string
	RequiredResources int
	Tentative         bool
}

func New(appGuid string, requiredResources int) Instance {
	return Instance{
		AppGuid:           appGuid,
		InstanceGuid:      util.NewGuid("INS"),
		RequiredResources: requiredResources,
		Tentative:         false,
	}
}
