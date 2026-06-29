// Package billing coordinates paid business flows across modules.
//
// Billing owns application-level orchestration such as settling captured payments,
// completing paid checkouts, renewing subscriptions, fulfilling subscription
// benefits, and applying usage credits.
//
// Billing should not become a CRUD module for customers, products, providers, or
// low-level records. Domain modules own their own state; billing owns the
// business flow across those modules.
package billing
