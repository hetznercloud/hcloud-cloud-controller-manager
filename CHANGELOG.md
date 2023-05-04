# Changelog

## v1.15.0

Affordable, sustainable & powerful! 🚀You can now get one of our Arm64 CAX servers to optimize your operations while minimizing your costs!
Discover Ampere’s efficient and robust Arm64 architecture and be ready to get blown away with its performance. 😎

Learn more: https://www.hetzner.com/news/arm64-cloud

### What's Changed
* fix(deps): update kubernetes packages to v0.26.3 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/404
* chore(deps): update golangci/golangci-lint docker tag to v1.52.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/405
* feat(helm): env var config by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/406
* chore(chart): basic README by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/407
* fix(chart): README typo by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/409
* chore(deps): update golangci/golangci-lint docker tag to v1.52.1 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/410
* chore(deps): update actions/stale action to v8 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/411
* chore(deps): update golangci/golangci-lint docker tag to v1.52.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/413
* refactor: Update & Fix golangci-lint by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/408
* feat: new dev/test environment by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/414
* fix(ci): run e2e tests on main by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/416
* feat(goreleaser): produce OCI manifest images by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/417
* feat: publish ARM container images by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/420


**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.14.2...v1.15.0

## v1.15.0-rc.0

### What's Changed
* fix(deps): update kubernetes packages to v0.26.3 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/404
* chore(deps): update golangci/golangci-lint docker tag to v1.52.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/405
* feat(helm): env var config by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/406
* chore(chart): basic README by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/407
* fix(chart): README typo by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/409
* chore(deps): update golangci/golangci-lint docker tag to v1.52.1 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/410
* chore(deps): update actions/stale action to v8 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/411
* chore(deps): update golangci/golangci-lint docker tag to v1.52.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/413
* refactor: Update & Fix golangci-lint by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/408
* feat: new dev/test environment by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/414
* fix(ci): run e2e tests on main by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/416
* feat(goreleaser): produce OCI manifest images by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/417


**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.14.2...v1.15.0-rc.0

## v1.14.2

### What's Changed
* chore: multiple improvements to the release process by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/394
* feat(helm): configurable image by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/396
* chore: README / comment cleanups by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/397
* feat(chart): metrics + PodMonitor support by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/399
* chore(deps): update actions/setup-go action to v4 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/400
* feat(chart): configurable cmdline args by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/401
* fix: handle nil servers in InstanceV2 #398 by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/402
* fix: many API requests from Routes controller by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/403


**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.14.1...v1.14.2

## v1.14.1

### What's Changed
* fix(ci): wrong version published when two tags point to same commit  by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/392


**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.14.0...v1.14.1

## v1.14.0

### The release pipeline for this version was broken and no Docker Image was actually published. Please use v1.14.1 instead.

### Notable Changes

* Significantly reduced the number of Requests made to the Hetzner Cloud API. While this does not solve all cases of API rate limits, the situation should be better than before.
  * feat: add InstancesV2 interface by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/385
  * refactor: unnecessary API call in instance reconciliation by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/386

### All Changes
* chore: add apricote as codeowner by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/335
* feat: test against Kubernetes v1.26 by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/334
* ci: stop logging api token by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/336
* chore: update codeowner to use groups by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/345
* fix(deploy): node.kubernetes.io/not-ready taint is NoExecute by @flokli in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/344
* Bump golang.org/x/text from 0.3.7 to 0.3.8 by @dependabot in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/346
* chore(deps): Bump golang.org/x/net from 0.0.0-20220225172249-27dd8689420f to 0.7.0 by @dependabot in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/352
* chore(deps): Bump github.com/emicklei/go-restful from 2.9.5+incompatible to 2.16.0+incompatible by @dependabot in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/343
* test(e2e): fix flake when LB health checks have not passed by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/354
* feat: drop support for Kubernetes v1.23 by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/350
* docs: fix skaffold guide by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/348
* docs(lb): reference existing lb #351 by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/353
* Configure Renovate by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/355
* chore(deps): update alpine docker tag to v3.17 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/357
* fix(deps): update module github.com/stretchr/testify to v1.8.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/356
* chore(deps): update golang docker tag to v1.20 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/359
* chore(deps): update golangci/golangci-lint docker tag to v1.51.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/360
* fix(deps): update module github.com/hetznercloud/hcloud-go to v1.40.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/361
* chore(deps): update actions/setup-go action to v3 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/367
* chore(deps): update docker/login-action action to v2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/369
* chore(deps): update docker/setup-buildx-action action to v2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/370
* chore(deps): update goreleaser/goreleaser-action action to v4 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/371
* fix(deps): update kubernetes packages to v0.26.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/365
* chore(deps): update actions/stale action to v7 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/368
* chore(deps): update actions/checkout action to v3 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/364
* fix(deps): update module golang.org/x/crypto to v0.6.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/366
* fix(deps): update module k8s.io/klog/v2 to v2.90.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/363
* chore: initial basic helm chart by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/375
* fix(deps): update module k8s.io/klog/v2 to v2.90.1 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/374
* chore(chart): resources configurable via values.yaml by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/376
* chore: basic .gitpod.yml by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/377
* fix(ci): main branch rename by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/379
* feat: add packaged helm chart to release artifacts by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/378
* refactor(e2e): remove dev-ccm manifests by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/380
* fix(deps): update module github.com/hetznercloud/hcloud-go to v1.41.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/383
* feat(ci): publish helm chart to repository by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/381
* fix(deps): update module golang.org/x/crypto to v0.7.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/382
* feat: add InstancesV2 interface by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/385
* fix: self-reported version not correct by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/387
* chore(ci): run e2e on public workers by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/388
* refactor: unnecessary API call in instance reconciliation by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/386
* test: use actual test cases by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/389
* ci: fix goreleaser helm chart config by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/390
* ci: build helm repo index by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/391

### New Contributors
* @apricote made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/335
* @renovate made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/355
* @samcday made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/375

**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.13.2...v1.14.0

## v1.14.0-rc.0

### Notable Changes

* Significantly reduced the number of Requests made to the Hetzner Cloud API. While this does not solve all cases of API rate limits, the situation should be better than before.
  * feat: add InstancesV2 interface by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/385
  * refactor: unnecessary API call in instance reconciliation by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/386

### What's Changed
* chore: add apricote as codeowner by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/335
* feat: test against Kubernetes v1.26 by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/334
* ci: stop logging api token by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/336
* chore: update codeowner to use groups by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/345
* fix(deploy): node.kubernetes.io/not-ready taint is NoExecute by @flokli in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/344
* Bump golang.org/x/text from 0.3.7 to 0.3.8 by @dependabot in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/346
* chore(deps): Bump golang.org/x/net from 0.0.0-20220225172249-27dd8689420f to 0.7.0 by @dependabot in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/352
* chore(deps): Bump github.com/emicklei/go-restful from 2.9.5+incompatible to 2.16.0+incompatible by @dependabot in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/343
* test(e2e): fix flake when LB health checks have not passed by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/354
* feat: drop support for Kubernetes v1.23 by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/350
* docs: fix skaffold guide by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/348
* docs(lb): reference existing lb #351 by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/353
* Configure Renovate by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/355
* chore(deps): update alpine docker tag to v3.17 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/357
* fix(deps): update module github.com/stretchr/testify to v1.8.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/356
* chore(deps): update golang docker tag to v1.20 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/359
* chore(deps): update golangci/golangci-lint docker tag to v1.51.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/360
* fix(deps): update module github.com/hetznercloud/hcloud-go to v1.40.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/361
* chore(deps): update actions/setup-go action to v3 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/367
* chore(deps): update docker/login-action action to v2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/369
* chore(deps): update docker/setup-buildx-action action to v2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/370
* chore(deps): update goreleaser/goreleaser-action action to v4 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/371
* fix(deps): update kubernetes packages to v0.26.2 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/365
* chore(deps): update actions/stale action to v7 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/368
* chore(deps): update actions/checkout action to v3 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/364
* fix(deps): update module golang.org/x/crypto to v0.6.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/366
* fix(deps): update module k8s.io/klog/v2 to v2.90.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/363
* chore: initial basic helm chart by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/375
* fix(deps): update module k8s.io/klog/v2 to v2.90.1 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/374
* chore(chart): resources configurable via values.yaml by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/376
* chore: basic .gitpod.yml by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/377
* fix(ci): main branch rename by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/379
* feat: add packaged helm chart to release artifacts by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/378
* refactor(e2e): remove dev-ccm manifests by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/380
* fix(deps): update module github.com/hetznercloud/hcloud-go to v1.41.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/383
* feat(ci): publish helm chart to repository by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/381
* fix(deps): update module golang.org/x/crypto to v0.7.0 by @renovate in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/382
* feat: add InstancesV2 interface by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/385
* fix: self-reported version not correct by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/387
* chore(ci): run e2e on public workers by @samcday in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/388
* refactor: unnecessary API call in instance reconciliation by @apricote in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/386

### New Contributors
* @apricote made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/335
* @renovate made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/355
* @samcday made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/375

**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.13.2...v1.14.0-rc.0

## v1.13.2

### What's Changed
* Fix PTR update for load balancer by @ym in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/315

### New Contributors
* @ym made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/315

**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.13.1...v1.13.2

## v1.13.1

### What's Changed
* Add skaffold for 1 click debugging + update k8s gitlab versions by @4ND3R50N in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/314
* Update hcloud go to v1.35.3 by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/317
* Fix goreleaser by @4ND3R50N in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/318


**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.13.0...v1.13.1

## v1.13.0

### What's Changed
* Update k8s dependencies to v0.20.13 by @fhofherr in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/260
* Use our own Runners by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/261
* feat: allow setting of reverse DNS records by @morremeyer in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/274
* Use Go 1.18 by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/293
* Update K8s/k3s Support Matrix by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/294
* Update Dependencies by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/295
* Update hcloud-go and fix possible crash cases for servers with flexib… by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/296
* Metrics for Hetzner API calls by @maksim-paskal in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/303
* Raise cache reload timeout limit by @4ND3R50N in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/309
* Update to Go 1.19 by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/311
* Prioritize IPv4 address family by dual-stack by @rastislavs in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/306
* Migrate to priorityClassName API by @onpaws in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/283
* Flag to disable network routes by @maksim-paskal in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/304
* Update all non k8s related dependencies to last versions by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/312
* Add support for k8s 1.25 by @LKaemmerling in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/313

### New Contributors
* @morremeyer made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/274
* @maksim-paskal made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/303
* @4ND3R50N made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/309
* @rastislavs made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/306
* @onpaws made their first contribution in https://github.com/hetznercloud/hcloud-cloud-controller-manager/pull/283

**Full Changelog**: https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.12.1...v1.13.0

## v1.12.1

### Changelog

* 1b33f524 Prepare Release v1.21.1
* 9fa68870 Update hcloud-go to v1.33 (#255)
* ff044e93 deploy: add missing operator: Exists (#251)
* 7c9948b6 Bump k8s.io/kubernetes from 1.18.3 to 1.18.19 (#243)
* 451703ae Testsetup: Unify with CSI Driver test setup suite (#244)
* 635cf10a Update docs (#240)
* f21278cc Health Check: Set healthcheck port to destination port if no port was defined via annotation (#239)


### Docker images

- `docker pull hetznercloud/hcloud-cloud-controller-manager:v1.12.1`

## v1.12.0

### Changelog

* 580c9db9 Prepare Release v1.12.0
* 9d259b9b Bring IPv6 flag in line with private ingress flag (#237)
* 7728df20 add explanation for private node IPs (#219)
* 3f1a081f Build and test with go 1.17 (#235)
* 867e2377 Ignore stale routes on RouteList (#238)
* fb6b551c Use Metadata Client provided by hcloud-go (#234)
* bcf0e74e Update README for kube-proxy IPVS information (#213)
* 6f30ee1d Update hcloud-go to v1.28.0


### Docker images

- `docker pull hetznercloud/hcloud-cloud-controller-manager:v1.12.0`

## v1.11.1

### Changelog

* b721e5ae fix release asset version


### Docker images

- `docker pull hetznercloud/hcloud-cloud-controller-manager:latest`
- `docker pull hetznercloud/hcloud-cloud-controller-manager:v1.11.1`

## v1.11.0

### Changelog

* 659f728c Use ::1 host of the IPv6 subnet as the instance address
* f8d6673c Support for IPv6 NodeAddresses
* 32f602a0 Apply review results
* 354f8f85 Add Master is attached to configured Network check on controller boot.
* 1e444837 Fix typo in log message (#207)
* bf44907b Improving documentation
* e52a79be Update README.md to include Networks support
* 0d3274ca Fix for typo in hcloud command
* 1641943b Fix glob for deployment yamls


### Docker images

- `docker pull hetznercloud/hcloud-cloud-controller-manager:latest`
- `docker pull hetznercloud/hcloud-cloud-controller-manager:v1.11.0`

## v1.10.0

### Changelog

* b54847b9 Add option to disable IPv6 for load balancers
* 13cac638 Add #190 to yaml templates


### Docker images

- `docker pull hetznercloud/hcloud-cloud-controller-manager:latest`
- `docker pull hetznercloud/hcloud-cloud-controller-manager:v1.10.0`

## v1.10.0-rc2

### Changelog

* f40fa216 Fix generation of deployment yamls


### Docker images

- `docker pull hetznercloud/hcloud-cloud-controller-manager:latest`
- `docker pull hetznercloud/hcloud-cloud-controller-manager:v1.10.0-rc2`

## v1.10.0-rc1

### Changelog

* a0e90cae Automate release process
* 96013341 Tolerate node-role.kubernetes.io/control-plane:NoSchedule taints


### Docker images

- `docker pull hetznercloud/hcloud-cloud-controller-manager:latest`
- `docker pull hetznercloud/hcloud-cloud-controller-manager:v1.10.0-rc1`

## v1.9.1

* Fix: add correct version number to config files

## v1.9.0

* Add support for setting load balancer values via cluster-wide defaults: `HCLOUD_LOAD_BALANCERS_LOCATION`, `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`, `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS`, `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP` (#125)
* Fix: allow referencing certificates by name (#116)
* Update build to go 1.16
* Update hcloud-go to 1.25.0
* Fix: Creating a Route may fail because of stale cache
* Add support for Hetzner Cloud Managed Certificates

## v1.8.1

* Fix: excessive calls to `/v1/servers` endpoint.

## v1.8.0

* Fix: nil pointer dereference when Load Balancers were disabled
* Update hcloud-go to 1.22.0
* Update build to go 1.15
* Fix: update default health check (#87)
* Fix: Ignore protected Load Balancers on deletion instead of raising an error

You can update by running
```
### for Networks Version
kubectl apply -f https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/ccm-networks.yaml

### for without Networks
https://raw.githubusercontent.com/hetznercloud/hcloud-cloud-controller-manager/master/deploy/ccm.yaml

```

## v1.7.0

* Fix: nil pointer dereference when Network was not found
* Update hcloud-go to 1.20.0
* Add `HCLOUD_LOAD_BALANCERS_ENABLED` env variable to disable the Load
  Balancer feature, per default the Load Balancers are enabled.
* Use defaults if health check annotations are not set.
* Add support for changing the Load Balancer type

## v1.6.1

* Add missing support Load Balancer sticky sessions
* Fix wrong parsing of health check timeout and interval

## v1.6.0

* Add support for hcloud Load Balancer
* Update kubernetes dependencies to v1.16.2
* Update build to go 1.14

You can find a detailed description for the new Load Balancers under https://github.com/hetznercloud/hcloud-cloud-controller-manager/blob/master/docs/load_balancers.md

## v1.5.1

- Add better error handling and validation for certain errors related to wrong API tokens

## v1.5.0

* Add Support for Kubernetes 1.16

## v1.4.0

* Add Networks Support


This release was tested on Kubernetes 1.15.x.

## v1.3.0

* Kubernetes 1.11 and 1.12 are now supported
* update hcloud-go to 1.12.0

## v1.2.0

- update hcloud-go to v1.4.0
- update kubernetes dependencies to v1.9.3

## v1.1.0

* update kubernetes dependencies to v1.9.2

## v1.0.0

* initial release
