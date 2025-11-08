# Chaos Mesh RFCs

This directory contains Request for Comments (RFC) documents for proposed features and significant changes to Chaos Mesh.

## What is an RFC?

An RFC (Request for Comments) is a design document that describes a new feature, change, or proposal for Chaos Mesh. RFCs help the community:

- Understand the motivation and use cases for new features
- Evaluate design alternatives
- Provide feedback before implementation begins
- Document architectural decisions

## RFC List

- [bpftime-userspace-chaos.md](./bpftime-userspace-chaos.md) - Proposal for adding userspace fault injection using bpftime/eBPF

## How to Contribute an RFC

1. Create a new markdown file in this directory with a descriptive name
2. Use the following structure:
   - **Summary**: Brief overview of the proposal
   - **Motivation**: Why this change is needed
   - **Proposed Solution**: Detailed design and implementation plan
   - **Alternatives Considered**: Other approaches that were evaluated
   - **References**: Links to related documentation
3. Submit a pull request for review
4. Engage with community feedback and iterate on the design

## RFC Status

RFCs can be in one of the following states:

- **Draft**: Initial proposal, under discussion
- **Accepted**: Approved for implementation
- **Implemented**: Feature has been completed
- **Rejected**: Proposal was not accepted
- **Superseded**: Replaced by a newer RFC

For questions about the RFC process, please refer to the [CONTRIBUTING.md](../CONTRIBUTING.md) or reach out to the maintainers.
