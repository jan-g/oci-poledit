package dg

import (
	"context"
	"strings"
	"github.com/jan-g/oci-poledit"
	"github.com/jan-g/oci-poledit/edit"
	"github.com/oracle/oci-go-sdk/identity"
)

type dg struct {
	identity.DynamicGroup
}

var _ poledit.Handler = &dg{}

func New() *dg {
	return &dg{}
}

func (p dg) Find(ctx context.Context, parent string, name string, iam identity.IdentityClient) (poledit.Handler, error) {
	pols, err := p.List(ctx, parent, iam)
	if err != nil {
		return nil, err
	}
	for _, p := range pols {
		if *p.Name == name {
			return &dg{p}, nil
		}
	}
	return &dg{}, poledit.NotFoundError
}

func (p *dg) List(ctx context.Context, parent string, iam identity.IdentityClient) ([]identity.DynamicGroup, error) {
	var page *string = nil
	res := []identity.DynamicGroup{}
	for {
		ps, err := iam.ListDynamicGroups(ctx, identity.ListDynamicGroupsRequest{
			CompartmentId: &parent,
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, ps.Items...)
		page = ps.OpcNextPage
		if page == nil {
			break
		}
	}
	return res, nil
}

func (p *dg) Edit() (poledit.Handler, error) {
	var mr []string = []string{*p.MatchingRule}
	lines, err := edit.Edit(mr)
	if err != nil {
		return nil, err
	}
	newMr := ""
	if len(lines) == 0 {
		newMr = ""
	} else if len(lines) == 1 {
		newMr = lines[0]
	} else {
		newMr = "any {" + strings.Join(lines, ",") + "}"
	}

	pp := &dg{p.DynamicGroup}
	pp.MatchingRule = &newMr

	return pp, nil
}

func (p *dg) JsonEdit() (poledit.Handler, error) {
	pol, err := edit.Json(&p.DynamicGroup)
	return &dg{*(pol.(*identity.DynamicGroup))}, err
}

func (p *dg) WithDescription(description *string) poledit.Handler {
	pp := &dg{p.DynamicGroup}
	pp.Description = description
	return pp
}

func (p *dg) WithName(name *string) poledit.Handler {
	pp := &dg{p.DynamicGroup}
	pp.Name = name
	return pp
}

func (p *dg) ShouldCreate() bool {
	return p.MatchingRule != nil && len(*p.MatchingRule) > 0
}

func (p *dg) DoCreate(iam identity.IdentityClient, compartment identity.Compartment) error {
	_, err := iam.CreateDynamicGroup(context.Background(), identity.CreateDynamicGroupRequest{
		CreateDynamicGroupDetails: identity.CreateDynamicGroupDetails{
			CompartmentId: compartment.Id,
			Name:          p.Name,
			MatchingRule:    p.MatchingRule,
			Description:   p.Description,
			FreeformTags:  p.FreeformTags,
			DefinedTags:   p.DefinedTags,
		},
	})
	return err
}

func (p *dg) DoUpdate(iam identity.IdentityClient) error {
	_, err := iam.UpdateDynamicGroup(context.Background(), identity.UpdateDynamicGroupRequest{
		DynamicGroupId: p.Id,
		UpdateDynamicGroupDetails: identity.UpdateDynamicGroupDetails{
			Description:  p.Description,
			MatchingRule:   p.MatchingRule,
			FreeformTags: p.FreeformTags,
			DefinedTags:  p.DefinedTags,
		},
	})
	return err
}

func (p *dg) DoDelete(iam identity.IdentityClient) error {
	_, err := iam.DeleteDynamicGroup(context.Background(), identity.DeleteDynamicGroupRequest{
		DynamicGroupId: p.Id,
	})
	return err
}
