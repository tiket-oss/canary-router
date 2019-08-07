[![Build Status](https://travis-ci.com/tiket-libre/canary-router.svg?branch=master)](https://travis-ci.com/tiket-libre/canary-router)
[![codecov](https://codecov.io/gh/tiket-libre/canary-router/branch/master/graph/badge.svg)](https://codecov.io/gh/tiket-libre/canary-router)


# Canary Router

Canary Router is a reverse proxy that forwards HTTP requests to one of the two endpoints based on arbitrary logic provided by additional "sidecar" server.

## Rationale

Rolling out new services in production environment can be quite a daunting task. New, non battle-tested services are prone to fail even though it has been tested, since there's no sure way we can replicate the variety that exists in production environment.

Several techniques have been proposed in order to deal with this problem. One of the most popular one is [Canary Release](https://martinfowler.com/bliki/CanaryRelease.html). Its main idea is to roll out the new services in small subsets of users before gradually release it to every users.

But there exists similar cases where we don't want to route traffic simply by dividing it by the users. Maybe we want to roll out new service with new technology/stack, but still holds the same contract/API. We don't really need to divide the traffics by the users. We can use different strategies such as:

- Limiting throughput; Maintain the number of request passed through the new service below certain threshold, any more than that gets routed back to old service.
- Circuit breaking; All traffics are routed to new services until it fails, any subsequent requests should be routed to the old services.

There are of course many other cases with many other strategies, but the bottom line is that all of these can be arbitrary and very specific.

### Solution

Canary Router acts as the proxy which sole purpose is to route traffic between two different endpoints. It relies on other service as the basis of its routing decision. We'll call this service *sidecar service* as it holds similarity to [Sidecar Pattern](https://docs.microsoft.com/en-us/azure/architecture/patterns/sidecar).

This sidecar service will hold the logic on how to route the traffic, and provide it for the Canary Router via REST endpoint. This endpoint will return responses based on a convention: return http status code OK (200) to route traffic to the primary/new service, and return No-Content (204) to route it to secondary/old service.

This way the Canary Router will be decoupled from any dependency that might occur as a result of the routing logic. The sidecar service on the other hand, are free to access external resource to determine where should the traffic be routed. Keep in mind that by doing so we might damage the application performance dramatically.

## Installation

To install the canary router, make sure that you have Go installed and run the following command.

```sh
go get -u -v github.com/tiket-libre/canary-router
```

## Usage

To run this, make sure you have the services involved defined in a JSON configuration file:

```json
{
    "listen-port": 1345,
    "main-target": "http://server-mono.localhost",
    "canary-target": "http://server-micro.localhost",
    "sidecar-url": "http://sidecar.localhost",
    "circuit-breaker": {
        "request-limit-canary": 300
    },
    "instrumentation": {
        "port": "8888"
    }
}
```

| Field                | Description                               | Type    |
| -------------------- | ----------------------------------------- | ------- |
| listen-port          | Port that are used to serve Canary Router | INTEGER |
| main-target          | URL of the old/secondary service          | STRING  |
| canary-target        | URL of the new/primary service            | STRING  |
| sidecar-url          | URL of the sidecar service                | STRING  |
| instrumentation.port | Port to access instrumentatione endpoint  | STRING  |

After filling out the configuration file, provide its path in the `-c` or `--config` flag to run the canary router:

```sh
canary-router -c config.json
```

## Instrumentation

*UNDER CONSTRUCTION*
