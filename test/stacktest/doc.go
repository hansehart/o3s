// Package stacktest raises parts of the o3s stack from this branch for the integration and e2e
// tests, the way httptest raises a server for HTTP tests.
//
// It drives the docker compose CLI with the daily stack and its dev overlay, which builds the
// images from this checkout, so a test exercises the code under review.
package stacktest
