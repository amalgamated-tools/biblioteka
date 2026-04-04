// Package jobs defines the background job handler functions registered with
// the asynq worker. Jobs cover library scanning (discovering new book files),
// individual file processing (metadata extraction, cover writing, sidecar
// creation), and bulk re-scanning of all configured libraries.
package jobs
