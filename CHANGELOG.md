# Changelog

## [v1.21.0](https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases/tag/v1.21.0)

### Feature Highlights &amp; Upgrade Notes

#### Load Balancer IPs set to Private IPs

If networking support is enabled, the load balancer IPs are now populated with the private IPs, unless the `load-balancer.hetzner.cloud/disable-private-ingress` annotation is set to `true`. Please make sure that you configured the annotation according to your needs, for example if you are using `external-dns`.

#### Provided-By Label

We introduced a the label `instance.hetzner.cloud/provided-by`, which will be automatically added to all **new** nodes. This label can have the values `cloud` or `robot` to distinguish between our products. We use this label in the csi-driver to ensure the daemonset is only running on cloud nodes. We recommend to add this label to your existing nodes with the appropriate value.

- `kubectl label node $CLOUD_NODE_NAME instance.hetzner.cloud/provided-by=cloud`
- `kubectl label node $ROBOT_NODE_NAME instance.hetzner.cloud/provided-by=robot`

#### Load Balancer IPMode Proxy

Kubernetes KEP-1860 added a new field to the Load Balancer Service Status that allows us to mark if the IP address we add should be considered as a Proxy (always send traffic here) and VIP (allow optimization by keeping the traffic in the cluster).

Previously Kubernetes considered all IPs as VIP, which caused issues when when the PROXY protocol was in use. We have previously recommended to use the annotation `load-balancer.hetzner.cloud/hostname` to workaround this problem.

We now set the new field to `Proxy` if the PROXY protocol is active so the issue should no longer appear. If you  only added the `load-balancer.hetzner.cloud/hostname` annotation for this problem, you can remove it after upgrading.

Further information:

- https://github.com/kubernetes/enhancements/issues/1860
- https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/160#issuecomment-788638132

### Features

- **service**: Specify private ip for loadbalancer (#724)
- add support &amp; tests for Kubernetes 1.31 (#747)
- **helm**: allow setting extra pod volumes via chart values  (#744)
- **instance**: add label to distinguish servers from Cloud and Robot (#764)
- emit event when robot server name and node name mismatch (#773)
- **load-balancer**: Set IPMode to &#34;Proxy&#34; if load balancer is configured to use proxy protocol (#727) (#783)
- **routes**: emit warning if cluster cidr is misconfigured (#793)
- **load-balancer**: ignore nodes that don&#39;t use known provider IDs (#780)
- drop tests for kubernetes v1.27 and v1.28

### Bug Fixes

- populate ingress private ip when disable-private-ingress is false (#715)
- wrong version logged on startup (#729)
- invalid characters in label instance-type of robot servers (#770)
- no events are emitted as broadcaster has no sink configured (#774)

### Kubernetes Support

This version was tested with Kubernetes 1.29 - 1.31. Furthermore, we dropped v1.27 and v1.28 support.

## [1.20.0](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.19.0...v1.20.0) (2024-07-08)


### Features

* add support & tests for Kubernetes 1.29 ([#600](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/600)) ([e8fabda](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/e8fabda9ab2e607bcb9a88a7e4e3454d10f1e2a0))
* add support & tests for Kubernetes 1.30 ([#679](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/679)) ([0748b6e](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/0748b6e4457227cea77c733b897ce63e0aa0da9b))
* drop tests for kubernetes v1.25 ([#597](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/597)) ([58261ec](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/58261ec84252da0291770095081fbf49c3e6f659))
* drop tests for kubernetes v1.26 ([#680](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/680)) ([9c4be01](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/9c4be01659d8ed2607c410639fa8719aedb22c2a))
* emit Kubernetes events for error conditions ([#598](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/598)) ([e8f9199](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/e8f9199975fe4a458f962a73caa4e4a7091093ee))
* **helm,manifests:** only specify container args instead of command ([#691](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/691)) ([2ba4058](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/2ba40588d3b3b44ac3c0fa4ff9ae9e9fd3336cc9))
* **helm:** allow setting affinity for deployment ([#686](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/686)) ([1a8ea95](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/1a8ea95571a0048c96160756b0d1c40f1a8a8b70))
* read HCLOUD_TOKEN from file ([#652](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/652)) ([a4343b8](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/a4343b84ea3fc6662f1f263f41325eea2e749c41))


### Bug Fixes

* **routes:** many requests for outdated routes by rate limiting ([#675](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/675)) ([e283b7d](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/e283b7deea83bc8bd9b20ad8d098884da3eda554))

## [1.19.0](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.18.0...v1.19.0) (2023-12-07)


### Features

* **chart:** add daemonset and node selector ([#537](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/537)) ([a94384f](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/a94384feb782529e4f0c2424fb037704f495fb83))
* **config:** stricter validation for settings `HCLOUD_LOAD_BALANCERS_ENABLED`, `HCLOUD_METRICS_ENABLED` & `HCLOUD_NETWORK_ROUTES_ENABLED` ([#546](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/546)) ([335a2c9](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/335a2c98e5ad1ca97e8e17e5eaebf2906cda8e60))
* **helm:** remove "v" prefix from chart version ([#565](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/565)) ([f11aa0d](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/f11aa0df8056e7c406fd214570e032820f0559d7)), closes [#529](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/529)
* **load-balancer:** handle planned targets exceedings max targets ([#570](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/570)) ([8bb131f](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/8bb131ff66dcd657b6d3e58f0937a7f266553667))
* remove unused variable NODE_NAME ([#545](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/545)) ([a659408](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/a65940830c4c92d53e55df9258a4bcc0a0a72abe))
* **robot:** handle ratelimiting with constant backoff ([#572](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/572)) ([2ddc201](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/2ddc201a6134f91a11e555d6fcbc2d2048b669a6))
* support for Robot servers ([#561](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/561)) ([65dea11](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/65dea11f93ce6ff413cea468b3c8d59487dde346))

## [1.19.0-rc.0](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.18.0...v1.19.0-rc.0) (2023-12-01)


### Features

* **chart:** add daemonset and node selector ([#537](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/537)) ([a94384f](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/a94384feb782529e4f0c2424fb037704f495fb83))
* **config:** stricter validation for settings `HCLOUD_LOAD_BALANCERS_ENABLED`, `HCLOUD_METRICS_ENABLED` & `HCLOUD_NETWORK_ROUTES_ENABLED` ([#546](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/546)) ([335a2c9](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/335a2c98e5ad1ca97e8e17e5eaebf2906cda8e60))
* **helm:** remove "v" prefix from chart version ([#565](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/565)) ([f11aa0d](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/f11aa0df8056e7c406fd214570e032820f0559d7)), closes [#529](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/529)
* **load-balancer:** handle planned targets exceedings max targets ([#570](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/570)) ([8bb131f](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/8bb131ff66dcd657b6d3e58f0937a7f266553667))
* remove unused variable NODE_NAME ([#545](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/545)) ([a659408](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/a65940830c4c92d53e55df9258a4bcc0a0a72abe))
* **robot:** handle ratelimiting with constant backoff ([#572](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/572)) ([2ddc201](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/2ddc201a6134f91a11e555d6fcbc2d2048b669a6))
* support for Robot servers ([#561](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/561)) ([65dea11](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/65dea11f93ce6ff413cea468b3c8d59487dde346))

## [1.18.0](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.17.2...v1.18.0) (2023-09-18)


### Features

* build with Go 1.21 ([#516](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/516)) ([7bf7e71](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/7bf7e7165ce5f603463ab7bbc0f623bff774aff0))
* **chart:** configure additional tolerations ([#518](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/518)) ([0d25cb6](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/0d25cb6cb5313b5ac82c1343de657e08255ef76a)), closes [#512](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/512)
* **chart:** support running multiple replicas with leader election ([4b18ee5](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/4b18ee55b7d7b3ad7df2ad14e88a56c7fc7bb1b6))
* **load-balancer:** Add new node-selector annotation ([#514](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/514)) ([db2e6dc](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/db2e6dc46a2aa7e691e2ecb125dc770dc8963799))
* test against kubernetes v1.28 and drop v1.24 ([#500](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/500)) ([3adf781](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/3adf78150e71081c6e2b3199b8aae5c21ff5bac2))

## [1.17.2](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.17.1...v1.17.2) (2023-08-18)


### Bug Fixes

* **deploy:** do not bind webhook port 10260 ([#495](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/495)) ([52c5f38](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/52c5f38836d7b98e81bd650f2b0f537242431f4c))

## [1.17.1](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.17.0...v1.17.1) (2023-07-19)


### Bug Fixes

* **deploy:** make last resource name configurable ([#477](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/477)) ([79ee405](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/79ee4051c2aff00c0977788e337ef6bbabe5eb92))
* **deploy:** manifests have wrong namespace "default" ([#476](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/476)) ([d800781](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/d8007810844a34aa2910fd7370febf3b2c79f0ab)), closes [#475](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/475)

## [1.17.0](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.16.0...v1.17.0) (2023-07-18)


### Features

* **helm:** allow to set labels and annotations for podMonitor ([#471](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/471)) ([5dad655](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/5dad655dd6f7091ea96ccbe6443f3f74a0d7c7ae))
* upgrade to hcloud-go v2 e4352ec  ([5a066a1](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/5a066a1825b1a10015bb481ccc164a65f508fe6d))


### Bug Fixes

* **helm-chart:** resource namespace and name ([#462](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/462)) ([0c4eee6](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/0c4eee63d2263cd4a6fa999da50b9c8734c4fa15))
* **routes:** deleting wrong routes when other server has same private IP ([#472](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/472)) ([5461038](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/546103879336e86ade1f2217f33003aa125bdb98)), closes [#470](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/470)

## [1.16.0](https://github.com/hetznercloud/hcloud-cloud-controller-manager/compare/v1.15.0-rc.0...v1.16.0) (2023-06-16)


### Features

* **helm:** allow to manually set network name or ID ([#458](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/458)) ([8410277](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/841027753b1ae140471a4bc862cad425daf725dc))


### Bug Fixes

* **ci:** qemu binfmt wrappers during release ([#421](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/421)) ([84a7541](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/84a754170eab6ed8f91035c84692d9cd82712254))
* **routes:** Only delete routes in the Cluster CIDR ([#432](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/432)) ([c35d292](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/c35d292b72003bd48203a6fa0fa476113633406a))


### Continuous Integration

* setup release-please ([#437](https://github.com/hetznercloud/hcloud-cloud-controller-manager/issues/437)) ([bbec89e](https://github.com/hetznercloud/hcloud-cloud-controller-manager/commit/bbec89ef9e1c6bf75b06dec4abdafc14afe549c9))

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
