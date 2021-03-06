// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

// +build go1.3

package lxdclient

import (
	"fmt"
	"time"

	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	coretesting "github.com/juju/juju/testing"
)

type imageSuite struct {
	testing.IsolationSuite
	Stub              *testing.Stub
	remoteWithTrusty  *stubRemoteClient
	remoteWithNothing *stubRemoteClient
}

var _ = gc.Suite(&imageSuite{})

func (s *imageSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)
	s.Stub = &testing.Stub{}
	s.remoteWithTrusty = &stubRemoteClient{
		stub: s.Stub,
		url:  "https://match",
		aliases: map[string]string{
			"trusty/amd64": "trusty-fingerprint",
		},
	}
	s.remoteWithNothing = &stubRemoteClient{
		stub:    s.Stub,
		url:     "https://missing",
		aliases: nil,
	}
}

type stubRemoteClient struct {
	stub    *testing.Stub
	url     string
	aliases map[string]string
}

var _ remoteClient = (*stubRemoteClient)(nil)

func (s *stubRemoteClient) URL() string {
	// Note we don't log calls to URL because they are not interesting, and
	// are generally just used for logging, etc.
	return s.url
}

func (s *stubRemoteClient) GetAlias(alias string) string {
	s.stub.AddCall("GetAlias", alias)
	if err := s.stub.NextErr(); err != nil {
		// GetAlias can't return an Err, but if we get an error, we'll
		// just treat that as a miss on the Alias lookup.
		return ""
	}
	return s.aliases[alias]
}

func (s *stubRemoteClient) CopyImage(imageTarget string, dest rawImageClient, aliases []string, callback func(string)) error {
	// We don't include the destination or the callback because they aren't
	// objects we can easily assert against.
	s.stub.AddCall("CopyImage", imageTarget, aliases)
	if err := s.stub.NextErr(); err != nil {
		return err
	}
	// This is to make this CopyImage act a bit like a real CopyImage. it
	// gives some Progress callbacks, and then sets the alias in the
	// target.
	if callback != nil {
		// The real one gives progress every 1%
		for i := 10; i <= 100; i += 10 {
			callback(fmt.Sprintf("%d%%", i))
			time.Sleep(1 * time.Microsecond)
		}
	}
	if stubDest, ok := dest.(*stubClient); ok {
		if stubDest.Aliases == nil {
			stubDest.Aliases = make(map[string]string)
		}
		for _, alias := range aliases {
			stubDest.Aliases[alias] = imageTarget
		}
	}
	return nil
}

func (s *stubRemoteClient) AsRemote() Remote {
	return Remote{
		Host:     s.url,
		Protocol: SimplestreamsProtocol,
	}
}

type stubConnector struct {
	stub          *testing.Stub
	remoteClients map[string]remoteClient
}

func MakeConnector(stub *testing.Stub, remotes ...remoteClient) *stubConnector {
	remoteMap := make(map[string]remoteClient)
	for _, remote := range remotes {
		remoteMap[remote.URL()] = remote
	}
	return &stubConnector{
		stub:          stub,
		remoteClients: remoteMap,
	}
}

func (s *stubConnector) connectToSource(remote Remote) (remoteClient, error) {
	s.stub.AddCall("connectToSource", remote.Host)
	if err := s.stub.NextErr(); err != nil {
		return nil, err
	}
	return s.remoteClients[remote.Host], nil
}

func (s *imageSuite) TestEnsureImageExistsAlreadyPresent(c *gc.C) {
	s.testEnsureImageExistsAlreadyPresent(c, "trusty", "ppc64el", "juju/trusty/ppc64el")
	s.testEnsureImageExistsAlreadyPresent(c, "centos7", "ppc64el", "juju/centos7/ppc64el")
}

func (s *imageSuite) testEnsureImageExistsAlreadyPresent(c *gc.C, series, arch, localAlias string) {
	connector := MakeConnector(s.Stub, s.remoteWithTrusty)
	raw := &stubClient{
		stub:    s.Stub,
		Aliases: map[string]string{localAlias: "dead-beef"},
	}
	client := &imageClient{
		raw:             raw,
		connectToSource: connector.connectToSource,
	}
	s.Stub.ResetCalls()
	image, err := client.EnsureImageExists(series, arch, []Remote{s.remoteWithTrusty.AsRemote()}, nil)
	c.Assert(err, jc.ErrorIsNil)
	c.Assert(image, gc.Equals, localAlias)
	s.Stub.CheckCalls(c, []testing.StubCall{
		{ // Check if we have the image already (we do)
			FuncName: "GetAlias",
			Args:     []interface{}{localAlias},
		},
	})
}

func (s *imageSuite) TestEnsureImageExistsFirstRemote(c *gc.C) {
	connector := MakeConnector(s.Stub, s.remoteWithTrusty)
	raw := &stubClient{
		stub: s.Stub,
		// We don't have the image locally
		Aliases: nil,
	}
	client := &imageClient{
		raw:             raw,
		connectToSource: connector.connectToSource,
	}
	remotes := []Remote{s.remoteWithTrusty.AsRemote()}
	s.Stub.ResetCalls()
	_, err := client.EnsureImageExists("trusty", "amd64", remotes, nil)
	c.Assert(err, jc.ErrorIsNil)
	// We didn't find it locally
	s.Stub.CheckCalls(c, []testing.StubCall{
		{ // Check if we have the image already (we don't)
			FuncName: "GetAlias",
			Args:     []interface{}{"juju/trusty/amd64"},
		},
		{ // We didn't so connect to the first remote
			FuncName: "connectToSource",
			Args:     []interface{}{"https://match"},
		},
		{ // And check if it has trusty (which it should)
			FuncName: "GetAlias",
			Args:     []interface{}{"trusty/amd64"},
		},
		{ // So Copy the Image
			FuncName: "CopyImage",
			Args:     []interface{}{"trusty/amd64", []string{"juju/trusty/amd64"}},
		},
	})
	// We've updated the aliases
	c.Assert(raw.Aliases, gc.DeepEquals, map[string]string{
		"juju/trusty/amd64": "trusty/amd64",
	})
}

func (s *imageSuite) TestEnsureImageExistsUnableToConnect(c *gc.C) {
	connector := MakeConnector(s.Stub, s.remoteWithTrusty)
	raw := &stubClient{
		stub: s.Stub,
		// We don't have the image locally
		Aliases: nil,
	}
	client := &imageClient{
		raw:             raw,
		connectToSource: connector.connectToSource,
	}
	badRemote := Remote{
		Host:     "https://nosuch-remote.invalid",
		Protocol: SimplestreamsProtocol,
	}
	s.Stub.ResetCalls()
	s.Stub.SetErrors(nil, errors.Errorf("unable-to-connect"))
	remotes := []Remote{badRemote, s.remoteWithTrusty.AsRemote()}
	_, err := client.EnsureImageExists("trusty", "amd64", remotes, nil)
	c.Assert(err, jc.ErrorIsNil)
	// We didn't find it locally
	s.Stub.CheckCalls(c, []testing.StubCall{
		{ // Check if we have the image already (we don't)
			FuncName: "GetAlias",
			Args:     []interface{}{"juju/trusty/amd64"},
		},
		{ // We didn't so connect to the first remote
			FuncName: "connectToSource",
			Args:     []interface{}{"https://nosuch-remote.invalid"},
		},
		{ // Connect failed to first, so connect to second and copy
			FuncName: "connectToSource",
			Args:     []interface{}{"https://match"},
		},
		{ // And check if it has trusty (which it should)
			FuncName: "GetAlias",
			Args:     []interface{}{"trusty/amd64"},
		},
		{ // So Copy the Image
			FuncName: "CopyImage",
			Args:     []interface{}{"trusty/amd64", []string{"juju/trusty/amd64"}},
		},
	})
	// We've updated the aliases
	c.Assert(raw.Aliases, gc.DeepEquals, map[string]string{
		"juju/trusty/amd64": "trusty/amd64",
	})
}

func (s *imageSuite) TestEnsureImageExistsNotPresentInFirstRemote(c *gc.C) {
	connector := MakeConnector(s.Stub, s.remoteWithNothing, s.remoteWithTrusty)
	raw := &stubClient{
		stub: s.Stub,
		// We don't have the image locally
		Aliases: nil,
	}
	client := &imageClient{
		raw:             raw,
		connectToSource: connector.connectToSource,
	}
	s.Stub.ResetCalls()
	remotes := []Remote{s.remoteWithNothing.AsRemote(), s.remoteWithTrusty.AsRemote()}
	_, err := client.EnsureImageExists("trusty", "amd64", remotes, nil)
	c.Assert(err, jc.ErrorIsNil)
	// We didn't find it locally
	s.Stub.CheckCalls(c, []testing.StubCall{
		{ // Lookup the alias locally
			FuncName: "GetAlias",
			Args:     []interface{}{"juju/trusty/amd64"},
		},
		{ // We didn't so connect to the first remote
			FuncName: "connectToSource",
			Args:     []interface{}{s.remoteWithNothing.URL()},
		},
		{ // Lookup the alias in the remote
			FuncName: "GetAlias",
			Args:     []interface{}{"trusty/amd64"},
		},
		{ // It wasn't found, so connect to second and look there
			FuncName: "connectToSource",
			Args:     []interface{}{s.remoteWithTrusty.URL()},
		},
		{ // And check if it has trusty (which it should)
			FuncName: "GetAlias",
			Args:     []interface{}{"trusty/amd64"},
		},
		{ // So Copy the Image
			FuncName: "CopyImage",
			Args:     []interface{}{"trusty/amd64", []string{"juju/trusty/amd64"}},
		},
	})
	// We've updated the aliases
	c.Assert(raw.Aliases, gc.DeepEquals, map[string]string{
		"juju/trusty/amd64": "trusty/amd64",
	})
}

func (s *imageSuite) TestEnsureImageExistsCallbackIncludesSourceURL(c *gc.C) {
	calls := make(chan string, 1)
	callback := func(message string) {
		select {
		case calls <- message:
		default:
		}
	}
	connector := MakeConnector(s.Stub, s.remoteWithTrusty)
	raw := &stubClient{
		stub: s.Stub,
		// We don't have the image locally
		Aliases: nil,
	}
	client := &imageClient{
		raw:             raw,
		connectToSource: connector.connectToSource,
	}
	remotes := []Remote{s.remoteWithTrusty.AsRemote()}
	_, err := client.EnsureImageExists("trusty", "amd64", remotes, callback)
	c.Assert(err, jc.ErrorIsNil)
	select {
	case message := <-calls:
		c.Check(message, gc.Matches, "copying image for juju/trusty/amd64 from https://match: \\d+%")
	case <-time.After(coretesting.LongWait):
		// The callbacks are made asynchronously, and so may not
		// have happened by the time EnsureImageExists exits.
		c.Fatalf("no messages received")
	}
}
