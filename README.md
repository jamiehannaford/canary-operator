# Canary Operator for Kubernetes

This is an operator for managing canary deployments on Kubernetes. It automates
the entire release process by progressively incrementing canary deployment size
over a set time period and a specified rate of increase.

Another potential use case is automating one's entire deployment pipeline. The
operator could monitor a registry for new image releases and add them to a release
queue automatically. This would facilitate things like nightly builds.

**This is still in the design phase. See the [spec](./spec.md) for details**.
