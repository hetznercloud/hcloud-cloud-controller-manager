Changelog
=========

master
------

* Fix: nil pointer dereference when Network was not found
* Update hcloud-go to 1.18.1
* Add `HCLOUD_LOAD_BALANCERS_ENABLED` env variable to disable the Load Balancer feature, per default the Load Balancers are enabled

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
