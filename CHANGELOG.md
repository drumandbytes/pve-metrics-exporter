# Changelog

## [1.2.1](https://github.com/drumandbytes/pve-metrics-exporter/compare/v1.2.0...v1.2.1) (2026-10-04)


### Performance Improvements

* cross-compile arm64 instead of building under QEMU ([#31](https://github.com/drumandbytes/pve-metrics-exporter/issues/31)) ([55b1d9d](https://github.com/drumandbytes/pve-metrics-exporter/commit/55b1d9d15bade32895936e101262b3c87717661d))

## [1.2.0](https://github.com/drumandbytes/pve-metrics-exporter/compare/v1.1.2...v1.2.0) (2026-09-28)


### Features

* ship Prometheus alerting rules with unit tests ([#28](https://github.com/drumandbytes/pve-metrics-exporter/issues/28)) ([f4d1fde](https://github.com/drumandbytes/pve-metrics-exporter/commit/f4d1fde8c080c09f0d36b04d3c753f9042cb2af9))

## [1.1.2](https://github.com/drumandbytes/pve-metrics-exporter/compare/v1.1.1...v1.1.2) (2026-09-20)


### Bug Fixes

* drop paths-ignore from the workflow hosting the required-check gate ([#19](https://github.com/drumandbytes/pve-metrics-exporter/issues/19)) ([f66378c](https://github.com/drumandbytes/pve-metrics-exporter/commit/f66378c3094a13833ee2077f7c0f05fac3fb6666))

## [1.1.1](https://github.com/drumandbytes/pve-metrics-exporter/compare/v1.1.0...v1.1.1) (2026-09-06)


### Bug Fixes

* **ci:** opt into app-token merge, so releases actually finish ([#13](https://github.com/drumandbytes/pve-metrics-exporter/issues/13)) ([4d9cf2f](https://github.com/drumandbytes/pve-metrics-exporter/commit/4d9cf2f2ef6108243ab0e39279b209998b1b977b))

## [1.1.0](https://github.com/drumandbytes/pve-metrics-exporter/compare/v1.0.0...v1.1.0) (2026-09-04)


### Features

* attach signed build provenance to published images ([#5](https://github.com/drumandbytes/pve-metrics-exporter/issues/5)) ([f45d951](https://github.com/drumandbytes/pve-metrics-exporter/commit/f45d95145af10bb39eafe43d684904a1588b8eef))


### Bug Fixes

* don't trigger a full image build+push for CI-config-only changes ([#3](https://github.com/drumandbytes/pve-metrics-exporter/issues/3)) ([dfae024](https://github.com/drumandbytes/pve-metrics-exporter/commit/dfae0240c18b4c0fce66bb3028eb04b4b3237750))
