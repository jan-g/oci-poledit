package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/jan-g/oci-poledit"
	"github.com/jan-g/oci-poledit/policy"
	"os"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"
)

// Usage: poledit compartment-name:compartment-name:policy-name

var (
	config          = flag.String("config", "~/.oci/config", "OCI configuration file")
	profile         = flag.String("profile", "DEFAULT", "profile to use")
	new             = flag.Bool("create", false, "create new policy")
	description     = flag.String("description", "", "description for new policy")
	json            = flag.Bool("json", false, "use json editor")
	tenancyOverride = flag.String("tenancy", "", "Override top-level tenancy")
)

var (
	policyHandler = policy.New()
)

func main() {
	flag.Parse()
	config, err := homedir.Expand(*config)
	if err != nil {
		panic(err)
	}
	if p, ok := os.LookupEnv("OCI_CLI_PROFILE"); ok && *profile == "DEFAULT" {
		profile = &p
	}
	provider, err := common.ConfigurationProviderFromFileWithProfile(config, *profile, "")
	if err != nil {
		panic(err)
	}
	tenancy, err := provider.TenancyOCID()
	if *tenancyOverride != "" {
		// Look up the profile first and try to use that
		tenancy = *tenancyOverride
		if tenancyProvider, err := common.ConfigurationProviderFromFileWithProfile(config, *tenancyOverride, ""); err == nil {
			if namedTenancy, err := tenancyProvider.TenancyOCID(); err == nil {
				tenancy = namedTenancy
			}
		}
	}
	if err != nil {
		panic(err)
	}
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Specify one path to a policy object")
	}
	iam, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		panic(err)
	}

	cp := CompartmentPath(flag.Arg(0))
	compartments := cp[:len(cp)-1]
	policyName := cp[len(cp)-1]

	compartment, err := ChainCompartmentLookup(context.Background(), iam, tenancy, compartments)
	if err != nil {
		panic(err)
	}

	policy, err := policyHandler.Find(context.Background(), *compartment.Id, policyName, iam)
	if err == nil {
		if *new {
			panic(errors.New("policy already exists"))
		}
		if *description != "" {
			policy = policy.WithDescription(description)
		}
		policy, err = editor(policy)
		if err != nil {
			panic(err)
		}

		if policy.ShouldCreate() {
			err := policy.DoUpdate(iam)
			if err != nil {
				panic(err)
			}
		} else {
			// Delete
			err := policy.DoDelete(iam)
			if err != nil {
				panic(err)
			}
		}
	} else if err == poledit.NotFoundError {
		if !*new {
			panic(errors.New("policy does not exist; use -create"))
		}
		if *description != "" {
			policy = policy.WithDescription(description)
		} else {
			panic(errors.New("use -description to specify a description"))
		}
		policy = policy.WithName(&policyName)

		policy, err = editor(policy)
		fmt.Printf("Policy is: %+v\n", policy)
		if err != nil {
			panic(err)
		}

		if policy.ShouldCreate() {
			err := policy.DoCreate(iam, compartment)
			if err != nil {
				panic(err)
			}
		}
	} else {
		panic(err)
	}
}

func CompartmentPath(path string) []string {
	cp := strings.Split(path, ":")
	for i, c := range cp {
		cp[i] = strings.Trim(c, " ")
	}
	return cp
}

func ChainCompartmentLookup(ctx context.Context, iam identity.IdentityClient, start string, rest []string) (identity.Compartment, error) {
	// Look up the top-level compartment
	t, err := iam.GetCompartment(ctx, identity.GetCompartmentRequest{
		CompartmentId: &start,
	})
	if err != nil {
		return identity.Compartment{}, err
	}

	comp := t.Compartment
	for _, part := range rest {
		c, err := FindCompartment(ctx, *comp.Id, part, iam)
		if err != nil {
			return identity.Compartment{}, err
		}
		comp = c
	}
	return comp, nil
}

func FindCompartment(ctx context.Context, parent string, name string, iam identity.IdentityClient) (identity.Compartment, error) {
	comps, err := ListCompartments(ctx, parent, iam)
	if err != nil {
		return identity.Compartment{}, err
	}
	for _, c := range comps {
		if *c.Name == name {
			return c, nil
		}
	}
	return identity.Compartment{}, poledit.NotFoundError
}

func ListCompartments(ctx context.Context, parent string, iam identity.IdentityClient) ([]identity.Compartment, error) {
	var page *string = nil
	res := []identity.Compartment{}
	for {
		cs, err := iam.ListCompartments(ctx, identity.ListCompartmentsRequest{
			CompartmentId: &parent,
			Page:          page,
		})
		if err != nil {
			return nil, err
		}
		res = append(res, cs.Items...)
		page = cs.OpcNextPage
		if page == nil {
			break
		}
	}
	return res, nil
}

func editor(handler poledit.Handler) (poledit.Handler, error) {
	editor := handler.Edit
	if *json {
		editor = handler.JsonEdit
	}
	return editor()
}
