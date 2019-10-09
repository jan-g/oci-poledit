# Quick and dirty policy-editing tool

Usage:

    % OCI_CLI_PROFILE=MY-POLICY ./poledit jang-sandbox:pol

You'll get dropped into an editor that holds the current set of statements, one per line.
Exiting the editor cleanly will cause the policy to be updated.

The path to a policy is given by `compartment-name:compartment-name:compartment-name:policy-name`.
