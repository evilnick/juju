// Copyright 2020 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package spaces

import (
	"github.com/juju/collections/set"
	"gopkg.in/juju/names.v3"
	"gopkg.in/mgo.v2/txn"

	"github.com/juju/juju/apiserver/common/networkingcommon"
	"github.com/juju/juju/controller"
	"github.com/juju/juju/core/constraints"
	"github.com/juju/juju/core/network"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/state"
)

// BlockChecker defines the block-checking functionality required by
// the spaces facade. This is implemented by apiserver/common.BlockChecker.
type BlockChecker interface {
	ChangeAllowed() error
	RemoveAllowed() error
}

// Address is an indirection for state.Address.
type Address interface {
	SubnetCIDR() string
}

// Machine defines the methods supported by a machine used in the space context.
type Machine interface {
	ApplicationNames() ([]string, error)
	AllAddresses() ([]Address, error)
	AllSpaces() (set.Strings, error)
}

// Constraints defines the methods supported by constraints used in the space context.
type Constraints interface {
	ID() string
	Value() constraints.Value
	ChangeSpaceNameOps(from, to string) []txn.Op
}

// Backing describes the state methods used in this package.
type Backing interface {
	environs.EnvironConfigGetter

	// ModelTag returns the tag of this model.
	ModelTag() names.ModelTag

	// SubnetByCIDR returns a unique subnet based on the input CIDR.
	SubnetByCIDR(cidr string) (networkingcommon.BackingSubnet, error)

	// MovingSubnet returns the subnet for the input ID,
	// suitable for moving to a new space.
	MovingSubnet(id string) (MovingSubnet, error)

	// AddSpace creates a space.
	AddSpace(Name string, ProviderId network.Id, Subnets []string, Public bool) error

	// AllSpaces returns all known Juju network spaces.
	AllSpaces() ([]networkingcommon.BackingSpace, error)

	// SpaceByName returns the Juju network space given by name.
	SpaceByName(name string) (networkingcommon.BackingSpace, error)

	// AllEndpointBindings loads all endpointBindings.
	AllEndpointBindings() ([]ApplicationEndpointBindingsShim, error)

	// AllMachines loads all machines.
	AllMachines() ([]Machine, error)

	// ApplyOperation applies a given ModelOperation to the model.
	ApplyOperation(state.ModelOperation) error

	// ControllerConfig returns the controller config.
	ControllerConfig() (controller.Config, error)

	// AllConstraints returns all constraints in the model.
	AllConstraints() ([]Constraints, error)

	// ConstraintsBySpaceName returns constraints found by spaceName.
	ConstraintsBySpaceName(name string) ([]Constraints, error)

	// SaveProviderSpaces loads providerSpaces into state.
	SaveProviderSpaces([]network.SpaceInfo) error

	// SaveProviderSubnets loads subnets into state.
	SaveProviderSubnets([]network.SubnetInfo, string) error

	// IsController returns true if this state instance
	// is for the controller model.
	IsController() bool
}
