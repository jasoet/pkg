// Package archtest mechanically enforces the library's v3 conventions.
// Tests here fail when a package's config struct loses its OTelConfig
// contract or a package drops its WithOTelConfig option. Extend the
// registries as packages are unified onto the conventions.
package archtest
