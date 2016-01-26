// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package client_test

import (
	"io"
	"net/http"

	"github.com/juju/errors"
	"github.com/juju/testing"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/resource"
	"github.com/juju/juju/resource/api"
	"github.com/juju/juju/resource/api/private"
	"github.com/juju/juju/resource/api/private/client"
	"github.com/juju/juju/resource/resourcetesting"
)

var _ = gc.Suite(&FacadeClientSuite{})

type FacadeClientSuite struct {
	testing.IsolationSuite

	stub *testing.Stub
	api  *stubAPI
}

func (s *FacadeClientSuite) SetUpTest(c *gc.C) {
	s.IsolationSuite.SetUpTest(c)

	s.stub = &testing.Stub{}
	s.api = &stubAPI{Stub: s.stub}
}

func (s *FacadeClientSuite) TestNewUnitFacadeClient(c *gc.C) {
	caller := &stubAPI{Stub: s.stub}
	doer := &stubAPI{Stub: s.stub}

	cl := client.NewUnitFacadeClient(caller, doer)

	s.stub.CheckNoCalls(c)
	aCaller, aDoer := client.ExposeClient(cl)
	c.Check(aCaller, gc.Equals, caller)
	c.Check(aDoer, gc.Equals, aDoer)
}

func (s *FacadeClientSuite) TestGetResource(c *gc.C) {
	opened := resourcetesting.NewResource(c, s.stub, "spam", "some data")
	s.api.setResource(opened.Resource, opened)
	cl := client.NewUnitFacadeClient(s.api, s.api)

	info, content, err := cl.GetResource("spam")
	c.Assert(err, jc.ErrorIsNil)

	s.stub.CheckCallNames(c, "Do", "FacadeCall")
	c.Check(info, jc.DeepEquals, opened.Resource)
	c.Check(content, jc.DeepEquals, opened)
}

type stubAPI struct {
	*testing.Stub

	ReturnFacadeCall private.ResourcesResult
	ReturnDo         *http.Response
}

func (s *stubAPI) setResource(info resource.Resource, reader io.ReadCloser) {
	s.ReturnFacadeCall = private.ResourcesResult{
		Resources: []private.ResourceResult{{
			Resource: api.Resource2API(info),
		}},
	}
	s.ReturnDo = &http.Response{
		Body: reader,
	}
}

func (s *stubAPI) FacadeCall(request string, params, response interface{}) error {
	s.AddCall("FacadeCall", params, response)
	if err := s.NextErr(); err != nil {
		return errors.Trace(err)
	}

	resp := response.(*private.ResourcesResult)
	*resp = s.ReturnFacadeCall
	return nil
}

func (s *stubAPI) Do(req *http.Request, body io.ReadSeeker, response interface{}) error {
	s.AddCall("Do", req, body, response)
	if err := s.NextErr(); err != nil {
		return errors.Trace(err)
	}

	resp := response.(**http.Response)
	*resp = s.ReturnDo
	return nil
}
