## What is a canary test?

This is best summed up by Google's [Site Reliability Engineering book](https://landing.google.com/sre/book/chapters/testing-reliability.html):

> The term canary comes from the phrase "canary in a coal mine," and refers to the practice of using a live bird to detect toxic gases before humans were poisoned. To conduct a canary test, a subset of servers is upgraded to a new version or configuration and then left in an incubation period. Should no unexpected variances occur, the release continues and the rest of the servers are upgraded in a progressive fashion. Should anything go awry, the single modified server can be quickly reverted to a known good state. We commonly refer to the incubation period for the upgraded server as "baking the binary."

## What problem is this trying to solve?

Canary tests are manually operated in the current state of Kubernetes. A
developer usually has to perform a rolling upgrade, monitor the state of the
canary, and then incrementally increase the proportion until the old version
is out of operation. This introduces an additional burden on the operator
since they have to perform multiple tasks. It also means that rollouts with
long-term incubation periods are usually not performed.

## What currently exists in the K8s ecosystem?

- Deployments allow rolling upgrades
- Services allow you to specify canaries and stable deployments that have a shared label
- Operator pattern  

## What will this do?

This operator will automate canary deployments according to a spec submitted
by a user. Over the course of a set duration, the operator will increase the
canary count according to a set rate (e.g. linear, cubic or exponential), and
perform smoke tests to catch regressions. After a set duration, the operator
will optionally delete the old deployment.

Another possible feature is the ability for the operator to manage a long-term
canary release process. Instead of targeting a specific release, the operator
could regularly scan a registry endpoint for new Docker images, and if a new
release is detected, it adds it to the release queue automatically. This would
facilitate automated nightly builds, for example.

## How will it do it?

The operator will create a Third Party Resource (TPR) using a spec. The spec
will contain the following fields:

- current deployment name
- new image tag
- total release timespan (time taken for canary to go from 0 to 100%)
- rate of increase (linear, cubic, exponential)
- initial canary pod count (defaults to the lowest value of either 10% of the current deployment or 1)
- delete old deployment?
- a command for monitoring overall liveness of service (HTTP, TCP or exec)
