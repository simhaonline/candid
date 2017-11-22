// Copyright 2017 Canonical Ltd.

package v1

import (
	"gopkg.in/juju/idmclient.v1/params"
	"gopkg.in/macaroon-bakery.v2/bakery"
	"gopkg.in/macaroon-bakery.v2/bakery/identchecker"

	"github.com/CanonicalLtd/blues-identity/internal/auth"
)

// opForRequest returns the operation that will be performed
// by the API handler method which takes the given argument r.
func opForRequest(r interface{}) bakery.Op {
	switch r := r.(type) {
	case *params.QueryUsersRequest:
		return auth.GlobalOp(auth.ActionRead)
	case *params.UserRequest:
		return auth.UserOp(r.Username, auth.ActionRead)
	case *params.SetUserRequest:
		if r.Owner != "" {
			return auth.UserOp(r.Owner, auth.ActionCreateAgent)
		}
		return auth.UserOp(r.Username, auth.ActionWriteAdmin)
	case *params.UserGroupsRequest:
		return auth.UserOp(r.Username, auth.ActionReadGroups)
	case *params.SetUserGroupsRequest:
		return auth.UserOp(r.Username, auth.ActionWriteGroups)
	case *params.ModifyUserGroupsRequest:
		return auth.UserOp(r.Username, auth.ActionWriteGroups)
	case *params.UserIDPGroupsRequest:
		return auth.UserOp(r.Username, auth.ActionReadGroups)
	case *params.WhoAmIRequest:
		return identchecker.LoginOp
	case *params.SSHKeysRequest:
		return auth.UserOp(r.Username, auth.ActionReadSSHKeys)
	case *params.PutSSHKeysRequest:
		return auth.UserOp(r.Username, auth.ActionWriteSSHKeys)
	case *params.DeleteSSHKeysRequest:
		return auth.UserOp(r.Username, auth.ActionWriteSSHKeys)
	case *params.UserTokenRequest:
		return auth.UserOp(r.Username, auth.ActionReadAdmin)
	case *params.VerifyTokenRequest:
		return auth.GlobalOp(auth.ActionVerify)
	case *params.UserExtraInfoRequest:
		return auth.UserOp(r.Username, auth.ActionReadAdmin)
	case *params.SetUserExtraInfoRequest:
		return auth.UserOp(r.Username, auth.ActionWriteAdmin)
	case *params.UserExtraInfoItemRequest:
		return auth.UserOp(r.Username, auth.ActionReadAdmin)
	case *params.SetUserExtraInfoItemRequest:
		return auth.UserOp(r.Username, auth.ActionWriteAdmin)
	case *params.DischargeTokenForUserRequest:
		return auth.GlobalOp(auth.ActionDischargeFor)
	default:
		logger.Infof("unknown API argument type %#v", r)
	}
	return bakery.Op{}
}
