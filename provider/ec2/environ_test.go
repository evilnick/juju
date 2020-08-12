// Copyright 2011, 2012, 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package ec2

import (
	"strings"

	gomock "github.com/golang/mock/gomock"
	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	amzec2 "gopkg.in/amz.v3/ec2"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/core/constraints"
	"github.com/juju/juju/core/network"
	"github.com/juju/juju/core/network/firewall"
	"github.com/juju/juju/environs"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/environs/context"
	"github.com/juju/juju/environs/simplestreams"
)

// Ensure EC2 provider supports the expected interfaces,
var (
	_ environs.NetworkingEnviron = (*environ)(nil)
	_ config.ConfigSchemaSource  = (*environProvider)(nil)
	_ simplestreams.HasRegion    = (*environ)(nil)
	_ context.Distributor        = (*environ)(nil)
)

type Suite struct{}

var _ = gc.Suite(&Suite{})

type RootDiskTest struct {
	series     string
	name       string
	constraint *uint64
	device     amzec2.BlockDeviceMapping
}

var commonInstanceStoreDisks = []amzec2.BlockDeviceMapping{{
	DeviceName:  "/dev/sdb",
	VirtualName: "ephemeral0",
}, {
	DeviceName:  "/dev/sdc",
	VirtualName: "ephemeral1",
}, {
	DeviceName:  "/dev/sdd",
	VirtualName: "ephemeral2",
}, {
	DeviceName:  "/dev/sde",
	VirtualName: "ephemeral3",
}}

func (*Suite) TestRootDiskBlockDeviceMapping(c *gc.C) {
	var rootDiskTests = []RootDiskTest{{
		"trusty",
		"nil constraint ubuntu",
		nil,
		amzec2.BlockDeviceMapping{VolumeSize: 8, DeviceName: "/dev/sda1"},
	}, {
		"trusty",
		"too small constraint ubuntu",
		pInt(4000),
		amzec2.BlockDeviceMapping{VolumeSize: 8, DeviceName: "/dev/sda1"},
	}, {
		"trusty",
		"big constraint ubuntu",
		pInt(20 * 1024),
		amzec2.BlockDeviceMapping{VolumeSize: 20, DeviceName: "/dev/sda1"},
	}, {
		"trusty",
		"round up constraint ubuntu",
		pInt(20*1024 + 1),
		amzec2.BlockDeviceMapping{VolumeSize: 21, DeviceName: "/dev/sda1"},
	}, {
		"win2012r2",
		"nil constraint windows",
		nil,
		amzec2.BlockDeviceMapping{VolumeSize: 40, DeviceName: "/dev/sda1"},
	}, {
		"win2012r2",
		"too small constraint windows",
		pInt(30 * 1024),
		amzec2.BlockDeviceMapping{VolumeSize: 40, DeviceName: "/dev/sda1"},
	}, {
		"win2012r2",
		"big constraint windows",
		pInt(50 * 1024),
		amzec2.BlockDeviceMapping{VolumeSize: 50, DeviceName: "/dev/sda1"},
	}, {
		"win2012r2",
		"round up constraint windows",
		pInt(50*1024 + 1),
		amzec2.BlockDeviceMapping{VolumeSize: 51, DeviceName: "/dev/sda1"},
	}}

	for _, t := range rootDiskTests {
		c.Logf("Test %s", t.name)
		cons := constraints.Value{RootDisk: t.constraint}
		mappings := getBlockDeviceMappings(cons, t.series, false)
		expected := append([]amzec2.BlockDeviceMapping{t.device}, commonInstanceStoreDisks...)
		c.Assert(mappings, gc.DeepEquals, expected)
	}
}

func pInt(i uint64) *uint64 {
	return &i
}

func (*Suite) TestPortsToIPPerms(c *gc.C) {
	testCases := []struct {
		about    string
		rules    firewall.IngressRules
		expected []amzec2.IPPerm
	}{{
		about: "single port",
		rules: firewall.IngressRules{firewall.NewIngressRule(network.MustParsePortRange("80/tcp"))},
		expected: []amzec2.IPPerm{{
			Protocol:  "tcp",
			FromPort:  80,
			ToPort:    80,
			SourceIPs: []string{"0.0.0.0/0"},
		}},
	}, {
		about: "multiple ports",
		rules: firewall.IngressRules{firewall.NewIngressRule(network.MustParsePortRange("80-82/tcp"))},
		expected: []amzec2.IPPerm{{
			Protocol:  "tcp",
			FromPort:  80,
			ToPort:    82,
			SourceIPs: []string{"0.0.0.0/0"},
		}},
	}, {
		about: "multiple port ranges",
		rules: firewall.IngressRules{
			firewall.NewIngressRule(network.MustParsePortRange("80-82/tcp")),
			firewall.NewIngressRule(network.MustParsePortRange("100-120/tcp")),
		},
		expected: []amzec2.IPPerm{{
			Protocol:  "tcp",
			FromPort:  80,
			ToPort:    82,
			SourceIPs: []string{"0.0.0.0/0"},
		}, {
			Protocol:  "tcp",
			FromPort:  100,
			ToPort:    120,
			SourceIPs: []string{"0.0.0.0/0"},
		}},
	}, {
		about: "source ranges",
		rules: firewall.IngressRules{firewall.NewIngressRule(network.MustParsePortRange("80-82/tcp"), "192.168.1.0/24", "0.0.0.0/0")},
		expected: []amzec2.IPPerm{{
			Protocol:  "tcp",
			FromPort:  80,
			ToPort:    82,
			SourceIPs: []string{"0.0.0.0/0", "192.168.1.0/24"},
		}},
	}}

	for i, t := range testCases {
		c.Logf("test %d: %s", i, t.about)
		ipperms := rulesToIPPerms(t.rules)
		c.Assert(ipperms, gc.DeepEquals, t.expected)
	}
}

// These Support checks are currently valid with a 'nil' environ pointer. If
// that changes, the tests will need to be updated. (we know statically what is
// supported.)
func (*Suite) TestSupportsNetworking(c *gc.C) {
	var env *environ
	_, supported := environs.SupportsNetworking(env)
	c.Assert(supported, jc.IsTrue)
}

func (*Suite) TestSupportsSpaces(c *gc.C) {
	callCtx := context.NewCloudCallContext()
	var env *environ
	supported, err := env.SupportsSpaces(callCtx)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(supported, jc.IsTrue)
	c.Check(environs.SupportsSpaces(callCtx, env), jc.IsTrue)
}

func (*Suite) TestSupportsSpaceDiscovery(c *gc.C) {
	var env *environ
	supported, err := env.SupportsSpaceDiscovery(context.NewCloudCallContext())
	// TODO(jam): 2016-02-01 the comment on the interface says the error should
	// conform to IsNotSupported, but all of the implementations just return
	// nil for error and 'false' for supported.
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(supported, jc.IsFalse)
}

func (*Suite) TestSupportsContainerAddresses(c *gc.C) {
	callCtx := context.NewCloudCallContext()
	var env *environ
	supported, err := env.SupportsContainerAddresses(callCtx)
	c.Assert(err, jc.Satisfies, errors.IsNotSupported)
	c.Assert(supported, jc.IsFalse)
	c.Check(environs.SupportsContainerAddresses(callCtx, env), jc.IsFalse)
}

func (*Suite) TestSelectSubnetIDsForZone(c *gc.C) {
	subnetZones := map[network.Id][]string{
		network.Id("bar"): {"foo"},
	}
	placement := network.Id("")
	az := "foo"

	var env *environ
	subnets, err := env.selectSubnetIDsForZone(subnetZones, placement, az)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(subnets, gc.DeepEquals, []network.Id{"bar"})
}

func (*Suite) TestSelectSubnetIDsForZones(c *gc.C) {
	subnetZones := map[network.Id][]string{
		network.Id("bar"): {"foo"},
		network.Id("baz"): {"foo"},
	}
	placement := network.Id("")
	az := "foo"

	var env *environ
	subnets, err := env.selectSubnetIDsForZone(subnetZones, placement, az)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(subnets, gc.DeepEquals, []network.Id{"bar", "baz"})
}

func (*Suite) TestSelectSubnetIDsForZoneWithPlacement(c *gc.C) {
	subnetZones := map[network.Id][]string{
		network.Id("bar"): {"foo"},
		network.Id("baz"): {"foo"},
	}
	placement := network.Id("baz")
	az := "foo"

	var env *environ
	subnets, err := env.selectSubnetIDsForZone(subnetZones, placement, az)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(subnets, gc.DeepEquals, []network.Id{"baz"})
}

func (*Suite) TestSelectSubnetIDsForZoneWithIncorrectPlacement(c *gc.C) {
	subnetZones := map[network.Id][]string{
		network.Id("bar"): {"foo"},
		network.Id("baz"): {"foo"},
	}
	placement := network.Id("boom")
	az := "foo"

	var env *environ
	_, err := env.selectSubnetIDsForZone(subnetZones, placement, az)
	c.Assert(err, gc.ErrorMatches, `subnets "boom" in AZ "foo" not found`)
}

func (*Suite) TestSelectSubnetIDForInstance(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	mockContext := NewMockProviderCallContext(ctrl)

	subnetZones := map[network.Id][]string{
		network.Id("some-sub"): {"some-az"},
		network.Id("baz"):      {"foo"},
	}
	placement := network.Id("")
	az := "foo"

	var env *environ
	subnet, err := env.selectSubnetIDForInstance(mockContext, false, subnetZones, placement, az)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(subnet, gc.DeepEquals, "baz")
}

func (*Suite) TestSelectSubnetIDForInstanceSelection(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	mockContext := NewMockProviderCallContext(ctrl)

	subnetZones := map[network.Id][]string{
		network.Id("baz"): {"foo"},
		network.Id("taz"): {"foo"},
	}
	placement := network.Id("")
	az := "foo"

	var env *environ
	subnet, err := env.selectSubnetIDForInstance(mockContext, false, subnetZones, placement, az)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(strings.HasSuffix(subnet, "az"), jc.IsTrue)
}

func (*Suite) TestSelectSubnetIDForInstanceWithNoMatchingZones(c *gc.C) {
	ctrl := gomock.NewController(c)
	defer ctrl.Finish()

	mockContext := NewMockProviderCallContext(ctrl)

	subnetZones := map[network.Id][]string{}
	placement := network.Id("")
	az := "invalid"

	var env *environ
	subnet, err := env.selectSubnetIDForInstance(mockContext, false, subnetZones, placement, az)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(subnet, gc.Equals, "")
}

func (*Suite) TestGetValidSubnetZoneMapOneSpaceConstraint(c *gc.C) {
	allSubnetZones := []map[network.Id][]string{
		{network.Id("sub-1"): {"az-1"}},
	}

	args := environs.StartInstanceParams{
		Constraints:    constraints.MustParse("spaces=admin"),
		SubnetsToZones: allSubnetZones,
	}

	subnetZones, err := getValidSubnetZoneMap(args)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(subnetZones, gc.DeepEquals, allSubnetZones[0])
}

func (*Suite) TestGetValidSubnetZoneMapOneBindingFanFiltered(c *gc.C) {
	allSubnetZones := []map[network.Id][]string{{
		network.Id("sub-1"):       {"az-1"},
		network.Id("sub-INFAN-2"): {"az-2"},
	}}

	args := environs.StartInstanceParams{
		SubnetsToZones: allSubnetZones,
		EndpointBindings: map[string]network.Id{
			"":    "space-1",
			"ep1": "space-1",
			"ep2": "space-1",
		},
	}

	subnetZones, err := getValidSubnetZoneMap(args)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(subnetZones, gc.DeepEquals, map[network.Id][]string{
		"sub-1": {"az-1"},
	})
}

func (*Suite) TestGetValidSubnetZoneMapNoIntersectionError(c *gc.C) {
	allSubnetZones := []map[network.Id][]string{
		{network.Id("sub-1"): {"az-1"}},
		{network.Id("sub-2"): {"az-2"}},
	}

	args := environs.StartInstanceParams{
		SubnetsToZones: allSubnetZones,
		Constraints:    constraints.MustParse("spaces=admin"),
		EndpointBindings: map[string]network.Id{
			"":    "space-1",
			"ep1": "space-1",
			"ep2": "space-1",
		},
	}

	_, err := getValidSubnetZoneMap(args)
	c.Assert(err, gc.ErrorMatches,
		`unable to satisfy supplied space requirements; spaces: \[admin\], bindings: \[space-1\]`)
}

func (*Suite) TestGetValidSubnetZoneMapIntersectionSelectsCorrectIndex(c *gc.C) {
	allSubnetZones := []map[network.Id][]string{
		{network.Id("sub-1"): {"az-1"}},
		{network.Id("sub-2"): {"az-2"}},
		{network.Id("sub-3"): {"az-2"}},
	}

	args := environs.StartInstanceParams{
		SubnetsToZones: allSubnetZones,
		Constraints:    constraints.MustParse("spaces=space-2,space-3"),
		EndpointBindings: map[string]network.Id{
			"":    "space-1",
			"ep1": "space-2",
			"ep2": "space-2",
		},
	}

	// space-2 is common to the bindings and constraints and is at index 1
	// of the sorted union.
	// This should result in the selection of the same index from the
	// subnets-to-zones map.

	subnetZones, err := getValidSubnetZoneMap(args)
	c.Assert(err, jc.ErrorIsNil)
	c.Check(subnetZones, gc.DeepEquals, allSubnetZones[1])
}
