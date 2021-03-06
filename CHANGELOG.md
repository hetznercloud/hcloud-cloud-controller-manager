Changelog
=========

v1.10.0
-------

*Note* starting from this release this file is no longer maintained. We
changed to an automated release process. Changelog entries can now be
found on the respective [GitHub
releases](https://github.com/hetznercloud/hcloud-cloud-controller-manager/releases).

v1.9.1
------

* Fix: add correct version number to config files

v1.9.0
------

* Add support for setting load balancer values via cluster-wide defaults: `HCLOUD_LOAD_BALANCERS_LOCATION`, `HCLOUD_LOAD_BALANCERS_NETWORK_ZONE`, `HCLOUD_LOAD_BALANCERS_DISABLE_PRIVATE_INGRESS`, `HCLOUD_LOAD_BALANCERS_USE_PRIVATE_IP` (#125)
* Fix: allow referencing certificates by name (#116)
* Update build to go 1.16
* Update hcloud-go to 1.25.0
* Fix: Creating a Route may fail because of stale cache
* Add support for Hetzner Cloud Managed Certificates

v1.8.1
------

* Fix: excessive calls to `/v1/servers` endpoint.

v1.8.0
------
* Fix: nil pointer dereference when Load Balancers were disabled
* Update hcloud-go to 1.22.0
* Update build to go 1.15
* Fix: update default health check (#87)
* Fix: Ignore protected Load Balancers on deletion instead of raising an error

v1.7.0
------

* Fix: nil pointer dereference when Network was not found
* Update hcloud-go to 1.20.0
* Add `HCLOUD_LOAD_BALANCERS_ENABLED` env variable to disable the Load
  Balancer feature, per default the Load Balancers are enabled.
* Use defaults if health check annotations are not set.
* Add support for changing the Load Balancer type

v1.6.1
------

* Add missing support Load Balancer sticky sessions
* Fix wrong parsing of health check timeout and interval

v1.6.0
------

* Add support for hcloud Load Balancer
* Update kubernetes dependencies to v1.16.2
* Update build to go 1.14

v1.5.2
------

* Fix nil pointer dereference if network does not exist anymore (#42).

v1.5.1
------

* Add better error handling and validation for certain errors related to wrong API tokens

v1.5.0
------

* Support for Kubernetes 1.16

v1.4.0
------

* Add Networks Support

v1.3.0
------

* Kubernetes 1.11 and 1.12 are now supported
* update hcloud-go to 1.12.0

v1.2.0
------

* update hcloud-go to v1.4.0
* update kubernetes dependencies to v1.9.3

v1.1.0
------

* update kubernetes dependencies to v1.9.2

v1.0.0
------

* initial release
