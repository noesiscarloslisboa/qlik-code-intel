# Native release implementation plan

1. Add native release jobs with a read-only validation stage and a narrowly
   scoped publication job. Support tag events and manual existing-tag backfills.
2. Add a standard-library packager, archive/checksum checks, license collection,
   and smoke tests against the extracted executable. Test validation failures.
3. Document downloads, manual backfills, tag releases, and platform requirements.
4. Run local Go checks, packaging tests, workflow lint, and a local archive smoke
   test. Push the workflow and run a non-publishing v0.1.0 build on every target.
5. Repair demonstrated portability failures, rerun native checks, then publish
   the tested binaries to the existing v0.1.0 release without moving its tag.
6. Verify public assets and checksums; record completed run evidence.
